const basePath = import.meta.env.BASE_URL.replace(/\/$/, "");

export const appBasename = basePath || "/";

export function apiPath(path: string): string {
  return `${basePath}/api${path}`;
}
