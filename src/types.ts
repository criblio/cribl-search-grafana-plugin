import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

/**
 * Query used with Cribl Search
 */
export interface CriblQuery extends DataQuery {
  /**
   * ID of the Cribl saved search
   */
  savedQueryId: string;
  /**
   * Max number of results to fetch
   */
  maxResults: number;
}

/**
 * Options configured for each CriblDataSource instance
 */
export interface CriblDataSourceOptions extends DataSourceJsonData {
  /**
   * Base URL to the Cribl organization/tenant site (i.e. https://main-elated-nash-dkf9udv.cribl-staging.cloud)
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
