import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

/**
 * Possible values of CriblQuery.type
 */
export type QueryType = 'adhoc' | 'saved';

/**
 * Query used with Cribl Search.  Can either use a saved search or run an adhoc query.
 */
export type CriblQuery = DataQuery & (
  {
    type: 'adhoc';
    /**
     * Ad-hoc query (Kusto)
     */
    query: string;
  } | {
    type: 'saved';
    /**
     * ID of the Cribl saved search
     */
    savedSearchId: string;
  }
);

/**
 * Options configured for each CriblDataSource instance
 */
export interface CriblDataSourceOptions extends DataSourceJsonData {
  /**
   * Base URL to the Cribl organization/tenant site (i.e. https://your-org-id.cribl.cloud)
   */
  criblOrgBaseUrl: string;
  /**
   * Client ID used to generate OAuth tokens
   */
  clientId?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface CriblSecureJsonData {
  /**
   * Client secret used to generate OAuth tokens
   */
  clientSecret?: string;
}
