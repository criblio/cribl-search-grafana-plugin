import {
  ArrayVector,
  DataFrame,
  DataQueryRequest,
  DataQueryResponse,
  DataSourceApi,
  DataSourceInstanceSettings,
  Field,
  FieldType,
} from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import { CriblQuery, CriblDataSourceOptions } from './types';

const QUERY_PAGE_SIZE = 1000;
const CRIBL_TIME_FIELD = '_time';

/**
 * Cribl Search DataSource
 */
export class CriblDataSource extends DataSourceApi<CriblQuery, CriblDataSourceOptions> {
  private authToken?: AuthToken;

  constructor(private readonly instanceSettings: DataSourceInstanceSettings<CriblDataSourceOptions>) {
    super(instanceSettings);
  }

  /**
   * Run a query, loading the most recent results for the given savedQueryId
   * @param options the options for this query
   * @returns the DataQueryResponse containing DataFrame(s)
   */
  async query(options: DataQueryRequest<CriblQuery>): Promise<DataQueryResponse> {
    // options.targets contains one or more queries, each of which results in a DataFrame
    return Promise.all(options.targets.map((q) => this.processQuery(q))).then((data) => ({ data }));
  }

  /**
   * Run a single query to produce a single DataFrame
   * @param criblQuery the query to run
   * @returns the DataFrame with the results
   */
  private async processQuery(criblQuery: CriblQuery): Promise<DataFrame> {
    let fields: Record<string, Field> = {};
    let eventCount = 0;
    let totalEventCount: number | undefined = undefined;
    let maxEventCount = criblQuery.maxResults ?? Number.MAX_VALUE;

    // Load the search results, paging through until we've hit maxResults or read them all, whatever comes first
    do {
      const response = await this.getAuthorization() // refresh auth token as needed
        .then((authorization) => lastValueFrom(getBackendSrv().fetch({
          method: 'GET',
          url: this.proxiedUrl('query'),
          headers: { ...authorization },
          responseType: 'text', // tell Grafana not to bother trying to JSON parse it, it's NDJSON
          params: {
            queryId: criblQuery.savedQueryId,
            offset: eventCount,
            limit: Math.min(maxEventCount, QUERY_PAGE_SIZE),
          },
        })));
      if (response.status !== 200) {
        throw new Error(`Unexpected response status (${response.status})`);
      }

      // Parse the NDJSON lines
      const data = response.data as string;
      let lines = data.split('\n');
      for (let lineIdx = 0; lineIdx < lines.length && eventCount < maxEventCount; ++lineIdx) {
        if (lines[lineIdx].length === 0) {
          continue;
        }
        const event = JSON.parse(lines[lineIdx]);

        // First NDJSON line is our "header"
        if (lineIdx === 0) {
          totalEventCount = event.totalEventCount;
          continue;
        }

        // Grab the keys and values from the event and append them to our DataFrame
        for (const key of Object.keys(event)) {
          const v: unknown = event[key];

          // Establish the Field
          let field = fields[key];
          if (!field) {
            field = fields[key] = {
              name: key === CRIBL_TIME_FIELD ? 'Time' : key, // Grafana uses "Time" as the standard time field name
              type: (!!v ? this.getFieldType(key, v) : null) as FieldType, // possibly null for now, we'll try to resolve it later
              values: new ArrayVector(),
              config: {},
            };

            // Backfill the value as undefined for any prior rows (since this field was absent until now)
            for (let i = 0; i < eventCount; ++i) {
              (field.values as ArrayVector).add(undefined);
            }
          }

          // Add the value
          (field.values as ArrayVector).add(key === CRIBL_TIME_FIELD ? this.timeToIsoString(v) : v);

          // If we've only seen null in this field so far, set the field type if we know it now
          if (field.type == null && !!v) {
            field.type = this.getFieldType(key, v);
          }

          // Track min/max if it's a number field
          if (typeof v === 'number') {
            if (field.config.min == null || v < field.config.min) {
              field.config.min = v;
            }
            if (field.config.max == null || v > field.config.max) {
              field.config.max = v;
            }
          }
        }

        ++eventCount;
      }
    } while (eventCount < maxEventCount && eventCount < (totalEventCount ?? 0));

    return {
      refId: criblQuery.refId,
      fields: Object.values(fields),
      length: eventCount,
    };
  }

  /**
   * Given a key and value from an event, get the corresponding Grafana FieldType
   * @param key the key (only used to see if it's "_time")
   * @param v the value
   * @returns the Grafana FieldType of the value
   */
  private getFieldType(key: string, v: unknown): FieldType {
    if (key === CRIBL_TIME_FIELD) {
      return FieldType.time;
    }
    switch (typeof v) {
      case 'number':
      case 'bigint':
        return FieldType.number;
      case 'string':
        return FieldType.string;
      case 'boolean':
        return FieldType.boolean;
      default:
        return FieldType.other; // i.e. object
    }
  }

  /**
   * Convert the value of the "_time" field to an ISO timestamp string
   * @param v the value
   * @returns an ISO timestamp string
   */
  private timeToIsoString<T>(v: T): string | T {
    switch (typeof v) {
      case 'number':
      case 'string':
        return new Date(+v * 1000).toISOString(); // our _time is in sec, Grafana Time needs to be ms
      default:
        return v; // kinda unexpected, but let's just pass it through as-is
    }
  }

  /**
   * Test the datasource, given its current configuration.  A successful test entails:
   * 1. Get an auth token.
   * 2. Hit the API with a simple HEAD request, ensuring the config & auth token are both valid.
   * @returns an object with status and message
   */
  async testDatasource(): Promise<{ status: string, message: string }> {
    return this.getAuthorization()
      .then((authorization) => lastValueFrom(getBackendSrv().fetch<any>({
        method: 'HEAD',
        url: this.proxiedUrl('testDatasource'),
        headers: { ...authorization },
      })))
      .then((_) => {
        return {
          status: 'success',
          message: 'Your Cribl data source is working properly.',
        };
      })
      .catch((err) => {
        return {
          status: 'error',
          message: JSON.stringify(err, Object.getOwnPropertyNames(err)),
        };
      });
  }

  /**
   * Get a valid, up-to-date Authorization header for use in API calls.  This function will cache and
   * refresh the auth token as needed.  It examines criblOrgBaseUrl to determine which environment
   * (production, staging, local) and invokes the respective authentication API in that environment.
   * @returns an object with "Authorization" key that can be used in "headers" on an API request
   */
  private async getAuthorization(): Promise<{ Authorization: string }> {
    // Refresh the token slightly earlier (30s) than strictly necessary
    if (!this.authToken || this.authToken.expiresAt <= (Date.now() + 30000)) {
      if (/cribl(-staging)?\.cloud$/.test(this.instanceSettings.jsonData.criblOrgBaseUrl)) {
        this.authToken = await this.refreshAuthTokenViaOAuth();
      } else {
        this.authToken = await this.refreshAuthTokenViaLocalAuth();
      }
    }
    return { Authorization: `Bearer ${this.authToken!.token}` };
  }

  /**
   * This is the typical way the plugin loads an access token when it's live (vs. local dev).
   * We hit the /oauth/token API, passing the configured clientId and clientSecret.
   * @returns the new AuthToken
   */
  private async refreshAuthTokenViaOAuth(): Promise<AuthToken> {
    const env = this.instanceSettings.jsonData.criblOrgBaseUrl.endsWith('cribl-staging.cloud') ? 'staging' : 'production';
    const response = await lastValueFrom(getBackendSrv().fetch<OAuthTokenResponse>({
      method: 'POST',
      url: this.proxiedUrl(`${env}Auth`),
    }));
    if (response.status === 200 && response.data?.access_token?.length > 0) {
      return {
        token: response.data.access_token,
        expiresAt: Date.now() + (response.data.expires_in * 1000),
      };
    } else {
      throw new Error(`Unexpected OAuth response: ${JSON.stringify(response, undefined, 2)}`);
    }
  }

  /**
   * For local dev, this uses the /auth/login API to get an auth token 
   * @returns the new AuthToken token
   */
  private async refreshAuthTokenViaLocalAuth(): Promise<AuthToken> {
    const response = await lastValueFrom(getBackendSrv().fetch<LocalLoginResponse>({
      method: 'POST',
      url: this.proxiedUrl('localAuth'),
    }));
    if (response.status !== 200) {
      throw new Error('Failed to acquire auth token');
    }
    return {
      token: response.data.token,
      expiresAt: parseJwtExp(response.data.token) * 1000,
    };
  }

  /**
   * Used for proxying requests through the Grafana backend, which is required when sending sensitive data
   * like clientId and clientSecret to the OAuth API.  We use "routes" defined in plugin.json.
   * https://grafana.com/docs/grafana/latest/developers/plugins/add-authentication-for-data-source-plugins/#add-a-proxy-route-to-your-plugin
   */
  private proxiedUrl(routePath: string): string {
    return `${this.instanceSettings.url}/${routePath}`;
  }
}

/**
 * Response we get back from /auth/login (for local dev)
 */
interface LocalLoginResponse {
  token: string,
}

/**
 * Response we get back from /oauth/token
 */
interface OAuthTokenResponse {
  access_token: string,
  scope: string,
  expires_in: number,
  token_type: 'Bearer',
}

/**
 * Auth token, cached until it needs to be refreshed
 */
interface AuthToken {
  token: string,
  expiresAt: number,
}

/**
 * Parse a JWT and get its expiration time
 * @param token the encoded JWT
 * @returns the expiration time in epoch seconds
 */
function parseJwtExp(token: string): number {
  const b64 = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
  const jsonPayload = decodeURIComponent(window.atob(b64).split('').map(function(c) {
      return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
  }).join(''));
  return JSON.parse(jsonPayload).exp;
}
