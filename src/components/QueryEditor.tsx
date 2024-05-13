import React, { ChangeEvent, useEffect, useRef, useState } from 'react';
import { InlineField, Input, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { CriblDataSource } from '../datasource';
import { CriblDataSourceOptions, CriblQuery } from '../types';
import { debounce } from 'lodash';

type Props = QueryEditorProps<CriblDataSource, CriblQuery, CriblDataSourceOptions>;

export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
  const { maxResults } = query;

  const debouncedOnRunQuery = useRef(debounce(onRunQuery, 750)).current;
  const [savedSearchIdOptions, setSavedSearchIdOptions] = useState<SelectableValue[]>([]);

  const onSavedQueryIdChange = (sv: SelectableValue<string>) => {
    const v = sv.value?.replace(/\s+/g, '') ?? ''; // auto-trim/remove any whitespace
    if (v.match(/^[a-zA-Z0-9_]+$/)) {
      onChange({ ...query, savedSearchId: v });
      debouncedOnRunQuery();
    }
  };

  const onMaxResultsChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, maxResults: Math.max(1, +event.target.value) });
    debouncedOnRunQuery();
  };

  // Load the saved search IDs, let the user just pick one
  useEffect(() => {
    const loadSavedSearchIds = async () => {
      try {
        const savedSearchIds = await datasource.loadSavedSearchIds();
        const options = [
          { label: '', value: '' },
          ...savedSearchIds.map((value) => ({ value, label: value })),
        ];
        setSavedSearchIdOptions(options);
      } catch (err) {
        console.log(`Failed to load saved search IDs: ${err}`);
      }
    };
    loadSavedSearchIds();
  }, [datasource]);

  return (
    <div className="gf-form">
      <InlineField label="Saved Search ID" labelWidth={24} tooltip="ID of the Cribl saved search">
        <Select onChange={onSavedQueryIdChange} options={savedSearchIdOptions} width={24} />
      </InlineField>
      <InlineField label="Max Results" labelWidth={24} tooltip="Max results to fetch">
        <Input onChange={onMaxResultsChange} value={maxResults ?? 1000} width={24} type="number" />
      </InlineField>
    </div>
  );
}
