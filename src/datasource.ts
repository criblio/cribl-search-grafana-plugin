import { DataSourceInstanceSettings, CoreApp, ScopedVars } from "@grafana/data";
import { DataSourceWithBackend, getTemplateSrv } from "@grafana/runtime";
import { CriblQuery, CriblDataSourceOptions, DEFAULT_QUERY } from "types";

export class CriblDataSource extends DataSourceWithBackend<CriblQuery, CriblDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<CriblDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<CriblQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(criblQuery: CriblQuery, scopedVars: ScopedVars) {
    switch (criblQuery.type) {
      case 'adhoc':
        return {
          ...criblQuery,
          // You can use dashboard variables in your query
          query: getTemplateSrv().replace(criblQuery.query, scopedVars),
        };
      case 'saved':
        return {
          ...criblQuery,
          // The savedSearchId can also be composed using dashboard variable(s)
          savedSearchId: getTemplateSrv().replace(criblQuery.savedSearchId, scopedVars),
        };
      default: // exhaustive check
        throw new Error(`Unexpected query type`);
    }
  }

  filterQuery(query: CriblQuery): boolean {
    return !query.hide && this.canRunQuery(query);
  }

  async loadSavedSearchIds() {
    return await this.getResource('savedSearchIds');
  }

  private canRunQuery(criblQuery: CriblQuery): boolean {
    switch (criblQuery.type) {
      case 'adhoc':
        return (criblQuery.query?.trim() ?? '').length > 0;
      case 'saved':
        return criblQuery.savedSearchId != null;
      default:
        return false;
    }
  }
}
