import { t } from "../i18n";

/** Error raised when the static data cannot be loaded. */
export class DataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DataError";
  }
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  return t("unexpectedError");
}
