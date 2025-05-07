import { useState, useEffect, useMemo, useCallback } from 'react';
import { GetData } from '../wailsjs/go/handler/Handler';
import { kibana } from '../wailsjs/go/models';
import Plot from 'react-plotly.js';
import { Data, Datum, Layout, PlotSelectionEvent } from 'plotly.js';

function App() {
  const [logs, setLogs] = useState<kibana.KibanaLog[] | undefined>(undefined);

  useEffect(() => {
    (async () => {
      try {
        const data = await GetData();
        setLogs(data.logs);
      } catch (err: unknown) {
        // Handle error?
      }
    })();
  }, []);

  const data: Data[] = useMemo(() => {
    const d = {
      x: [] as number[],
      y: [] as number[],
      text: [] as string[],
      customdata: [] as Datum[],
      mode: 'markers' as const,
      type: 'scatter' as const,
      hoverinfo: 'text' as const,
    };
    if (!logs) {
      return [d];
    }
    for (let i = 0; i < logs.length; i++) {
      const log = logs[i];
      d.x.push(log.coordinates.error.X);
      d.y.push(log.coordinates.error.Y);
      d.text.push(`${log._id}<br>${log._source.microservice}<br>${log._source.message}`);
      d.customdata.push(i);
    }
    return [d];
  }, [logs]);

  const layout: Partial<Layout> = useMemo(() => {
    return {
      dragmode: 'select',
      xaxis: {
        showticklabels: false,
        zeroline: false,
      },
      yaxis: {
        showticklabels: false,
        zeroline: false,
      },
    };
  }, []);

  const handleSelected = useCallback(
    (event: Readonly<PlotSelectionEvent>) => {
      console.log(event.points.length);
    },
    [logs]
  );

  return (
    <div id="App">
      <Plot
        data={data}
        layout={layout}
        onSelected={handleSelected}
        useResizeHandler={true}
        style={{ width: '100%', height: '100%' }}
      />
    </div>
  );
}

export default App;
