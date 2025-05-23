import { useCallback } from 'react';
import { LogFieldSelectorsActive } from '../../models/models';
import { KibanaErrorLog } from '../../models/kibana';

export type SelectedProps = {
  selecting: boolean;
  selectors: LogFieldSelectorsActive;
  logs: KibanaErrorLog[];
};

export const Selected: React.FC<SelectedProps> = ({ selecting, selectors, logs }) => {
  const renderField = useCallback((property: string, text: string) => {
    return (
      <div className="text-xs break-all text-left">
        <p className="text-red-500">{property}:</p>
        <p>{text}</p>
      </div>
    );
  }, []);

  const renderLog = useCallback(
    (log: KibanaErrorLog) => {
      return (
        <div key={log._id} className="text-sm">
          {selectors.id && renderField('id', log._id)}
          {selectors.microservice && renderField('microservice', log._source.microservice)}
          {selectors.message && renderField('message', log._source.message)}
          {selectors.errorMessage && renderField('errorMessage', log._source.errorMessage)}
        </div>
      );
    },
    [selectors, renderField]
  );

  return (
    <aside className="w-full h-full overflow-y-auto min-w-[100px] max-w-[250px]">
      <h2>Selected: {logs.length}</h2>
      {selecting && <p>Selecting...</p>}
      {!selecting && <div className="flex flex-col gap-4">{logs.map(renderLog)}</div>}
    </aside>
  );
};
