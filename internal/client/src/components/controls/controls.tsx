import { useCallback } from 'react';
import { LogFieldSelectors, LogFieldSelectorsActive } from '../../models/models';

export type ControlsProps = {
  selectors: LogFieldSelectorsActive;
  filters: {
    startDate: string;
    endDate: string;
  };
  onSelectorToggled(name: LogFieldSelectors, toggled: boolean): void;
  onFilterChanged(name: string, value: string): void;
};

export const Controls: React.FC<ControlsProps> = ({ filters, selectors, onSelectorToggled, onFilterChanged }) => {
  const handleSelectorToggle = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onSelectorToggled(event.target.name as LogFieldSelectors, event.target.checked);
    },
    [onSelectorToggled]
  );

  const handleFilterChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onFilterChanged(event.target.name, event.target.value);
    },
    [onFilterChanged]
  );

  const renderSelector = useCallback(
    (name: string, checked: boolean) => {
      return (
        <div className="flex w-full justify-between items-center">
          <label htmlFor={`log-field-${name}`}>{name}</label>
          <input
            id={`log-field-${name}`}
            type="checkbox"
            name={name}
            onChange={handleSelectorToggle}
            checked={checked}
          />
        </div>
      );
    },
    [handleSelectorToggle]
  );

  return (
    <aside className="w-full min-w-[100px] max-w-[250px]">
      <div>
        <h2>Labels</h2>
        <div className="flex flex-col p-2 gap-2">
          {renderSelector('id', selectors.id)}
          {renderSelector('microservice', selectors.microservice)}
          {renderSelector('message', selectors.message)}
          {renderSelector('errorMessage', selectors.errorMessage)}
        </div>
      </div>
      <div>
        <h2>Filters</h2>
        <div className="flex flex-col p-2 gap-2">
          <div className="flex w-full justify-between items-center">
            <label htmlFor={`log-filter-start`}>Start Date</label>
            <div className="border">
              <input
                id={`log-filter-start`}
                type="date"
                name="startDate"
                value={filters.startDate}
                onChange={handleFilterChange}
              />
            </div>
          </div>
          <div className="flex w-full justify-between items-center">
            <label htmlFor={`log-filter-end`}>End Date</label>
            <div className="border">
              <input
                id={`log-filter-end`}
                type="date"
                name="endDate"
                value={filters.endDate}
                onChange={handleFilterChange}
              />
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
};
