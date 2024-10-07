import { DataSourcePlugin } from '@grafana/data';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { CriblDataSourceOptions, CriblQuery } from 'types';
import { CriblDataSource } from 'datasource';

export const plugin = new DataSourcePlugin<CriblDataSource, CriblQuery, CriblDataSourceOptions>(CriblDataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
