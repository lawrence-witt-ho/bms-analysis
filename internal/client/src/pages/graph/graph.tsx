import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { LogFieldSelectors, LogFieldSelectorsActive } from '../../models/models';
import { KibanaErrorLog } from '../../models/kibana';
import { Data, Datum, Layout, PlotSelectionEvent, PlotType } from 'plotly.js';
import { Controls } from '../../components/controls/controls';
import Plot from 'react-plotly.js';
import { Selected } from '../../components/selected/selected';

type LogSelector = (log: KibanaErrorLog) => string;

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

const getDate = (d: Date) => {
  return new Intl.DateTimeFormat('en-CA', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(d);
};

export const Graph: React.FC<{ logs: KibanaErrorLog[]; clearLogs: () => void }> = ({ logs, clearLogs }) => {
  const [selectors, setSelectors] = useState<LogFieldSelectorsActive>({
    id: true,
    microservice: false,
    message: false,
    errorMessage: false,
  });
  const [filters, setFilters] = useState({
    startDate: getDate(new Date(new Date().setMonth(new Date().getMonth() - 1))),
    endDate: getDate(new Date()),
  });
  const [filteredLogs, setFilteredLogs] = useState<KibanaErrorLog[]>([]);
  const [selected, setSelected] = useState<KibanaErrorLog[]>([]);
  const [selecting, setSelecting] = useState(false);

  const selectingTimeout = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    setFilteredLogs(
      logs.filter((log) => {
        const timestamp = new Date(log._source['@timestamp']).getTime();
        const start = new Date(filters.startDate).getTime();
        const end = new Date(filters.endDate).getTime();
        const inRange = timestamp >= start && timestamp <= end;
        return inRange;
      })
    );
  }, [logs, filters]);

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
    for (let i = 0; i < filteredLogs.length; i++) {
      const log = filteredLogs[i];
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
  }, [selectors, filteredLogs]);

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

  const handleFilterChanged = useCallback((name: string, value: string) => {
    setFilters((f) => ({ ...f, [name]: value }));
    setSelected([]);
  }, []);

  const handleSelecting = useCallback(
    (event: Readonly<PlotSelectionEvent>) => {
      setSelecting(true);
      setSelected(event.points.flatMap((p) => (typeof p.customdata === 'number' ? filteredLogs[p.customdata] : [])));
      if (selectingTimeout.current != null) {
        clearTimeout(selectingTimeout.current);
      }
      selectingTimeout.current = setTimeout(() => {
        setSelecting(false);
        selectingTimeout.current = null;
      }, 250);
    },
    [filteredLogs]
  );

  const handleDeselect = useCallback(() => {
    setSelected([]);
  }, []);

  return (
    <div className="flex h-screen">
      <Controls
        filters={filters}
        selectors={selectors}
        onSelectorToggled={handleSelectorToggled}
        onFilterChanged={handleFilterChanged}
      />
      <div className="w-full flex flex-col">
        <div className="flex justify-between align-items px-5">
          <h2>
            {filteredLogs.length} of {logs.length} Logs
          </h2>
          <button className="cursor-pointer border px-1" onClick={clearLogs}>
            Eject File
          </button>
        </div>
        <Plot
          data={plotData}
          layout={plotLayout}
          onSelecting={handleSelecting}
          onDeselect={handleDeselect}
          useResizeHandler={true}
          style={{ width: '100%', height: '100%' }}
        />
      </div>
      <Selected selecting={selecting} selectors={selectors} logs={selected} />
    </div>
  );
};
