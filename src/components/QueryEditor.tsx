import React, { ChangeEvent, KeyboardEvent, useEffect, useRef, useState } from 'react';
import { InlineField, Select, TextArea } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { CriblDataSource } from '../datasource';
import { CriblDataSourceOptions, CriblQuery } from '../types';
import { debounce } from 'lodash';

type Props = QueryEditorProps<CriblDataSource, CriblQuery, CriblDataSourceOptions>;

const QUERY_TYPE_OPTIONS = ['saved', 'adhoc'].map((value) => ({ label: value, value }));
const DEFAULT_QUERY_TYPE = 'adhoc';

export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
  const debouncedOnRunQuery = useRef(debounce(onRunQuery, 750)).current;
  const [savedSearchIdOptions, setSavedSearchIdOptions] = useState<SelectableValue[]>([]);
  const [savedSearchId, setSavedSearchId] = useState('');

  const [queryType, setQueryType] = useState(query.type as string ?? DEFAULT_QUERY_TYPE);
  const [adhocQuery, setAdhocQuery] = useState(query.type === 'adhoc' ? query.query : '');

  const onQueryTypeChange = (sv: SelectableValue<string>) => {
    const newQueryType = sv.value ?? DEFAULT_QUERY_TYPE;
    setQueryType(newQueryType);
    if (newQueryType === 'saved') {
      onChange({ ...query, type: 'saved', savedSearchId });
    } else {
      onChange({ ...query, type: 'adhoc', query: adhocQuery });
    }
  };

  const onAdhocQueryChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const newQuery = event.target.value;
    setAdhocQuery(newQuery);
    onChange({ ...query, type: 'adhoc', query: newQuery });
    // Don't run it automatically, the user can hit Enter or click "Run query" when ready
  };

  const onAdhocQueryKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Enter' && !event.shiftKey) { // allow shift-enter to add a line break
      event.preventDefault();
      if (adhocQuery.trim().length > 0) {
        onRunQuery();
      }
    }
  };

  const onSavedQueryIdChange = (sv: SelectableValue<string>) => {
    const newSavedSearchId = sv.value?.replace(/\s+/g, '') ?? ''; // auto-trim/remove any whitespace
    setSavedSearchId(newSavedSearchId);
    if (newSavedSearchId !== savedSearchId && newSavedSearchId.match(/^[a-zA-Z0-9_]+$/)) {
      onChange({ ...query, type: 'saved', savedSearchId: newSavedSearchId });
      debouncedOnRunQuery();
    }
  };

  // Load the saved search IDs, let the user just pick one
  useEffect(() => {
    const loadSavedSearchIds = async () => {
      try {
        const savedSearchIds = await datasource.loadSavedSearchIds();
        setSavedSearchIdOptions([
          { label: 'Please select...', value: '' },
          ...savedSearchIds.map((value) => ({ value, label: value })),
        ]);
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
        <TextArea
          onChange={onAdhocQueryChange}
          onKeyDown={onAdhocQueryKeyDown}
          value={adhocQuery}
          rows={1}
          cols={72}
          type="string"
          placeholder='Enter your query, i.e. dataset="cribl_search_sample" | limit 42'
        />
      </InlineField>;
    }
  };

  return (
    <div className="gf-form">
      <InlineField label="Query Type" labelWidth={12}>
        <Select onChange={onQueryTypeChange} options={QUERY_TYPE_OPTIONS} value={queryType} width={12} />
      </InlineField>
      {getQueryUI()}
    </div>
  );
}
