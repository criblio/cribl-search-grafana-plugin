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
    if (criblQuery.type !== 'adhoc') {
      return criblQuery;
    }
    return {
      ...criblQuery,
      query: getTemplateSrv().replace(criblQuery.query, scopedVars),
    };
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
