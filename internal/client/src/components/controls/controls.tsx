import { useCallback } from 'react';
import { LogFieldSelectors, LogFieldSelectorsActive } from '../../models/models';

export type ControlsProps = {
  selectors: LogFieldSelectorsActive;
  onSelectorToggled(name: LogFieldSelectors, toggled: boolean): void;
};

export const Controls: React.FC<ControlsProps> = ({ selectors, onSelectorToggled }) => {
  const handleSelectorToggle = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onSelectorToggled(event.target.name as LogFieldSelectors, event.target.checked);
    },
    [onSelectorToggled]
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
    <aside className="w-full min-w-[100px] max-w-[200px]">
      <h2>Labels</h2>
      <div className="p-2 gap-2">
        {renderSelector('id', selectors.id)}
        {renderSelector('microservice', selectors.microservice)}
        {renderSelector('message', selectors.message)}
        {renderSelector('errorMessage', selectors.errorMessage)}
      </div>
    </aside>
  );
};
