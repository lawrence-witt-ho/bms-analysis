import { useCallback, useState } from 'react';
import { KibanaErrorLog } from './models/kibana';
import { Graph } from './pages/graph/graph';
import { Upload } from './pages/upload/upload';

function App() {
  const [logs, setLogs] = useState<KibanaErrorLog[] | undefined>(undefined);

  const handleLogsAdded = useCallback((logs: KibanaErrorLog[]) => {
    setLogs(logs);
  }, []);

  const handleLogsCleared = useCallback(() => {
    setLogs(undefined);
  }, []);

  return (
    <>
      {!logs && <Upload setLogs={handleLogsAdded} />}
      {logs && <Graph logs={logs} clearLogs={handleLogsCleared} />}
    </>
  );
}

export default App;
