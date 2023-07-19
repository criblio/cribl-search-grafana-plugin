import { DataSourcePlugin } from '@grafana/data';
import { CriblDataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { CriblQuery, CriblDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<CriblDataSource, CriblQuery, CriblDataSourceOptions>(CriblDataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
