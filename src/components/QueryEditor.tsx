import React, { ChangeEvent, KeyboardEvent, useEffect, useRef, useState } from 'react';
import { InlineField, Input, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { CriblDataSource } from '../datasource';
import { CriblDataSourceOptions, CriblQuery } from '../types';
import { debounce } from 'lodash';

type Props = QueryEditorProps<CriblDataSource, CriblQuery, CriblDataSourceOptions>;

const queryTypeOptions = ['saved', 'adhoc'].map((value) => ({ label: value, value }));

export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
  const debouncedOnRunQuery = useRef(debounce(onRunQuery, 750)).current;
  const [savedSearchIdOptions, setSavedSearchIdOptions] = useState<SelectableValue[]>([]);
  const [savedSearchId, setSavedSearchId] = useState('');

  const [queryType, setQueryType] = useState('saved');
  const [adhocQuery, setAdhocQuery] = useState('');

  const onQueryTypeChange = (sv: SelectableValue<string>) => {
    const newQueryType = sv.value ?? 'saved';
    setQueryType(newQueryType);
    if (newQueryType === 'saved') {
      onChange({ ...query, type: 'saved', savedSearchId });
    } else {
      onChange({ ...query, type: 'adhoc', query: adhocQuery });
    }
  };

  const onAdhocQueryChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newQuery = event.target.value;
    setAdhocQuery(newQuery);
    onChange({ ...query, type: 'adhoc', query: newQuery });
    // Don't run it automatically, the user can hit Enter or click "Run query" when ready
  };

  const onAdhocQueryKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Enter') {
      onRunQuery();
    }
  };

  const onSavedQueryIdChange = (sv: SelectableValue<string>) => {
    const newSavedSearchId = sv.value?.replace(/\s+/g, '') ?? ''; // auto-trim/remove any whitespace
    setSavedSearchId(newSavedSearchId);
    if (newSavedSearchId.match(/^[a-zA-Z0-9_]+$/)) {
      onChange({ ...query, type: 'saved', savedSearchId: newSavedSearchId });
      debouncedOnRunQuery();
    }
  };

  // Load the saved search IDs, let the user just pick one
  useEffect(() => {
    const loadSavedSearchIds = async () => {
      try {
        const savedSearchIds = await datasource.loadSavedSearchIds();
        setSavedSearchIdOptions(['', ...savedSearchIds].map((value) => ({ value, label: value })));
      } catch (err) {
        console.log(`Failed to load saved search IDs: ${err}`);
      }
    };
    loadSavedSearchIds();
  }, [datasource]);

  const getQueryUI = () => {
    if (queryType === 'saved') {
      return <InlineField label="Saved Search" labelWidth={16} tooltip="ID of the Cribl saved search">
        <Select onChange={onSavedQueryIdChange} options={savedSearchIdOptions} value={savedSearchId} width={24} />
      </InlineField>;
    } else {
      return <InlineField label="Query" labelWidth={10} tooltip="Cribl Search query (Kusto)">
        <Input onChange={onAdhocQueryChange} onKeyDown={onAdhocQueryKeyDown} value={adhocQuery} width={64} type="string" />
      </InlineField>;
    }
  };

  return (
    <div className="gf-form">
      <InlineField label="Query Type" labelWidth={12}>
        <Select onChange={onQueryTypeChange} options={queryTypeOptions} value={queryType} width={12} />
      </InlineField>
      {getQueryUI()}
    </div>
  );
}
