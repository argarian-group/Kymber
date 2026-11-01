const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

export function formatDate(value: string) {
  return dateFormat.format(new Date(value));
}
