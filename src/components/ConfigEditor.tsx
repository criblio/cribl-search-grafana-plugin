import React, { ChangeEvent, useState } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { CriblDataSourceOptions, CriblSecureJsonData } from 'types';

interface Props extends DataSourcePluginOptionsEditorProps<CriblDataSourceOptions, CriblSecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;

  const onChangeCriblOrgBaseUrl = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        criblOrgBaseUrl: parseCriblOrgBaseUrl(event.target.value),
      },
    });
  };
  const onResetCriblOrgBaseUrl = () => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        criblOrgBaseUrl: '',
      },
    });
  };

  const onChangeClientId = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        clientId: event.target.value,
      },
    });
  };
  const onResetClientId = () => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        clientId: '',
      },
    });
  };

  const onChangeClientSecret = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        clientSecret: event.target.value,
      },
    });
  };
  const onResetClientSecret = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        clientSecret: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        clientSecret: '',
      },
    });
  };

  const [queryTimeoutValidationError, setQueryTimeoutValidationError] = useState<string | null>(null);
  const onChangeQueryTimeoutSec = (event: ChangeEvent<HTMLInputElement>) => {
    const validateQueryTimeout = (value: string | undefined): boolean => {
      if (value != null && value.length > 0) {
        const queryTimeoutSec = +value;
        if (Number.isNaN(queryTimeoutSec) || queryTimeoutSec <= 0) {
          setQueryTimeoutValidationError(`Invalid query timeout (${value}), must be a positive integer`);
          return false;
        }
      }
      setQueryTimeoutValidationError(null);
      return true;
    };
    if (validateQueryTimeout(event.target.value)) {
      onOptionsChange({
        ...options,
        jsonData: {
          ...options.jsonData,
          queryTimeoutSec: event.target.value?.length > 0 ? +event.target.value : undefined,
        },
      });
    }
  };

  const { jsonData, secureJsonFields } = options;
  const secureJsonData = (options.secureJsonData || {}) as CriblSecureJsonData;

  return (
    <>
      <InlineField label="Cribl Organization URL" labelWidth={24}>
        <Input
          value={jsonData.criblOrgBaseUrl}
          placeholder="enter your Cribl URL, i.e. https://your-org-id.cribl.cloud"
          width={54}
          onReset={onResetCriblOrgBaseUrl}
          onChange={onChangeCriblOrgBaseUrl}
        />
      </InlineField>
      <InlineField label="Cribl Client ID" labelWidth={24}>
        <Input
          value={jsonData.clientId}
          placeholder="enter your Cribl Client ID"
          width={54}
          onReset={onResetClientId}
          onChange={onChangeClientId}
        />
      </InlineField>
      <InlineField label="Cribl Client Secret" labelWidth={24}>
        <SecretInput
          isConfigured={(secureJsonFields && secureJsonFields.clientSecret) as boolean}
          value={secureJsonData.clientSecret ?? ''}
          placeholder="enter your Cribl Client Secret"
          width={54}
          onReset={onResetClientSecret}
          onChange={onChangeClientSecret}
        />
      </InlineField>
      <InlineField label="Query Timeout" labelWidth={24}
        invalid={!!queryTimeoutValidationError}
        error={queryTimeoutValidationError}
        tooltip="How long (seconds) you're willing to wait for a query before we give up and cancel it.  Leave blank to mean no limit.">
        <Input
          value={jsonData.queryTimeoutSec ?? ''}
          placeholder="number of seconds (or blank for no timeout)"
          width={54}
          onChange={onChangeQueryTimeoutSec}
        />
      </InlineField>
    </>
  );
}

/**
 * Given any Cribl organization URL, parse it and return the base URL we need for API access
 * @param url any Cribl org URL, with or without a path (it's ignored)
 * @returns a Cribl org base URL for use with the API
 */
function parseCriblOrgBaseUrl(url: string): string {
  // This regex allows the user to paste any cloud URL for their org.  We pluck the tenant and domain and strip the rest.
  let match = url.match(/^https:\/\/([^./]+)\.(cribl(?:-gov)?(?:-staging)?\.cloud)(?:\/.*)$/);
  if (!match) {
    return url; // it's some other URL format (i.e. local dev), just use it as-is
  }

  const tenant = match[1];
  const domain = match[2];

  // The API is only accessible at <workspace>-<tenant>, so we need to ensure we add the workspace if it's not already there.
  match = tenant.match(/^([^-]+)-([^-]+-[^-]+-[^-]+)$/); // see if there's already a workspace
  const workspace = match ? match[1] : 'main'; // default to "main" if not
  const orgId = match ? match[2] : tenant;
  return `https://${workspace}-${orgId}.${domain}`;
}
