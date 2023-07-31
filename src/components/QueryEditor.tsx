import React, { ChangeEvent, useRef } from 'react';
import { InlineField, Input } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { CriblDataSource } from '../datasource';
import { CriblDataSourceOptions, CriblQuery } from '../types';
import { debounce } from 'lodash';

type Props = QueryEditorProps<CriblDataSource, CriblQuery, CriblDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const debouncedOnRunQuery = useRef(debounce(onRunQuery, 750)).current;

  const onSavedQueryIdChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, savedQueryId: event.target.value });
    debouncedOnRunQuery();
  };

  const onMaxResultsChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, maxResults: Math.max(1, +event.target.value) });
    debouncedOnRunQuery();
  };

  const { savedQueryId, maxResults } = query;

  return (
    <div className="gf-form">
      <InlineField label="Saved Search ID" labelWidth={24} tooltip="ID of the Cribl saved search">
        <Input onChange={onSavedQueryIdChange} value={savedQueryId ?? ''} width={24} type="string" />
      </InlineField>
      <InlineField label="Max Results" labelWidth={24} tooltip="Max results to fetch">
        <Input onChange={onMaxResultsChange} value={maxResults ?? 1000} width={24} type="number" />
      </InlineField>
    </div>
  );
}
