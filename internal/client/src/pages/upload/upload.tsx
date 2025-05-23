import { ChangeEvent, useCallback } from 'react';
import { KibanaErrorLog } from '../../models/kibana';

export const Upload: React.FC<{ setLogs: (logs: KibanaErrorLog[]) => void }> = ({ setLogs }) => {
  const handleFileInput = useCallback(
    (ev: ChangeEvent<HTMLInputElement>) => {
      const files = ev.target.files;
      if (!files) {
        return;
      }
      const file = files[0];
      if (!file) {
        return;
      }
      const reader = new FileReader();
      reader.onload = (pe) => {
        setLogs(JSON.parse(`${pe.target?.result}`));
      };
      reader.readAsText(file);
    },
    [setLogs]
  );

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="border p-2">
        <input type="file" onChange={handleFileInput} accept="application/JSON" />
      </div>
    </div>
  );
};
