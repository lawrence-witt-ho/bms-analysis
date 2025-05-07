import { useState, useEffect, useMemo, useCallback } from 'react';
import { GetData } from '../wailsjs/go/handler/Handler';
import { kibana } from '../wailsjs/go/models';
import Plot from 'react-plotly.js';
import { Data, Datum, Layout, PlotSelectionEvent, PlotType } from 'plotly.js';
import { LogFieldSelectors, LogFieldSelectorsActive } from './models/models';
import { Controls } from './components/controls/controls';
import { Selected } from './components/selected/selected';

type LogSelector = (log: kibana.KibanaLog) => string;

type PlotData = {
  x: number[];
  y: number[];
  text: string[];
  customdata: Datum[];
  mode: 'markers';
  type: PlotType;
  hoverinfo: 'text';
};

const logSelectors: { [K in LogFieldSelectors]: LogSelector } = {
  id: (log) => log._id,
  microservice: (log) => log._source.microservice,
  message: (log) => log._source.message,
  errorMessage: (log) =>
    `<br>${log._source.errorMessage.replace(new RegExp(`(.{1,${50}})(\\s+|$)`, 'g'), '$1<br>')}<br>`,
};

function App() {
  const [selectors, setSelectors] = useState<LogFieldSelectorsActive>({
    id: true,
    microservice: false,
    message: false,
    errorMessage: false,
  });
  const [logs, setLogs] = useState<kibana.KibanaLog[] | undefined>(undefined);
  const [selected, setSelected] = useState<kibana.KibanaLog[]>([]);
  const [selecting, setSelecting] = useState(false);

  useEffect(() => {
    (async () => {
      const data = await GetData();
      setLogs(data.logs);
    })();
  }, []);

  const plotData: Data[] = useMemo(() => {
    const d: PlotData = {
      x: [],
      y: [],
      text: [],
      customdata: [],
      mode: 'markers',
      type: 'scatter',
      hoverinfo: 'text',
    };
    if (!logs) {
      return [d];
    }
    for (let i = 0; i < logs.length; i++) {
      const log = logs[i];
      d.x.push(log.coordinates.error.X);
      d.y.push(log.coordinates.error.Y);
      d.text.push(
        Object.entries(selectors)
          .flatMap(([f, a]) => (a ? logSelectors[f as LogFieldSelectors](log) : []))
          .join('<br>')
      );
      d.customdata.push(i);
    }
    return [d];
  }, [selectors, logs]);

  const plotLayout: Partial<Layout> = useMemo(() => {
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

  const handleSelectorToggled = useCallback((name: LogFieldSelectors, checked: boolean) => {
    setSelectors((s) => ({ ...s, [name]: checked }));
  }, []);

  const handleSelecting = useCallback(
    (event: Readonly<PlotSelectionEvent>) => {
      if (!logs) {
        return;
      }
      setSelecting(true);
      setSelected(event.points.flatMap((p) => (typeof p.customdata === 'number' ? logs[p.customdata] : [])));
    },
    [logs]
  );

  const handleSelected = useCallback(() => {
    console.log('finished');
    setSelecting(false);
  }, []);

  const handleDeselect = useCallback(() => {
    setSelected([]);
  }, []);

  return (
    <div className="flex h-screen">
      <Controls selectors={selectors} onSelectorToggled={handleSelectorToggled} />
      <div className="w-full flex flex-col">
        <Plot
          data={plotData}
          layout={plotLayout}
          onSelecting={handleSelecting}
          onSelected={handleSelected}
          onDeselect={handleDeselect}
          useResizeHandler={true}
          style={{ width: '100%', height: '100%' }}
        />
      </div>
      <Selected selecting={selecting} selectors={selectors} logs={selected} />
    </div>
  );
}

export default App;
