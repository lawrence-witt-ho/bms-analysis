export const LOG_FIELD_SELECTORS = ['id', 'microservice', 'message', 'errorMessage'] as const;

export type LogFieldSelectors = (typeof LOG_FIELD_SELECTORS)[number];

export type LogFieldSelectorsActive = { [K in LogFieldSelectors]: boolean };
