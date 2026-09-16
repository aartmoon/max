// Only the official SDK accesses the MAX host. Launch data stays in memory and
// is never trusted as authentication until its signature is checked by backend.
export interface LaunchData {
  raw: string;
  startParam: string;
  platform: string;
}
interface MaxWebApp {
  initData?: string;
  initDataUnsafe?: { start_param?: string };
  platform?: string;
  ready?: () => void;
  BackButton?: {
    show(): void;
    hide(): void;
    onClick(callback: () => void): void;
    offClick(callback: () => void): void;
  };
  enableClosingConfirmation?: () => void;
  disableClosingConfirmation?: () => void;
}
declare global {
  interface Window {
    WebApp?: MaxWebApp;
  }
}
export interface MaxBridge {
  platform: "web" | "max";
  ready(): void;
  launchData(): LaunchData;
  backButton(callback: (() => void) | null): () => void;
  closingConfirmation(enabled: boolean): void;
}
export class BrowserBridge implements MaxBridge {
  platform = "web" as const;
  ready() {}
  launchData(): LaunchData {
    return { raw: "", startParam: "", platform: "browser" };
  }
  backButton() {
    return () => {};
  }
  closingConfirmation() {}
}
export class NativeMaxBridge implements MaxBridge {
  platform = "max" as const;
  private isReady = false;
  constructor(private app: MaxWebApp) {}
  ready() {
    if (this.isReady) return;
    this.isReady = true;
    // Some SDK versions initialize automatically and do not expose ready().
    this.app.ready?.();
  }
  launchData(): LaunchData {
    return {
      raw: this.app.initData ?? "",
      startParam: this.app.initDataUnsafe?.start_param ?? "",
      platform: this.app.platform ?? "unknown",
    };
  }
  backButton(callback: (() => void) | null) {
    const button = this.app.BackButton;
    if (!button) return () => {};
    if (!callback) {
      button.hide();
      return () => {};
    }
    button.onClick(callback);
    button.show();
    return () => {
      button.offClick(callback);
      button.hide();
    };
  }
  closingConfirmation(enabled: boolean) {
    if (enabled) this.app.enableClosingConfirmation?.();
    else this.app.disableClosingConfirmation?.();
  }
}
export const browserBridge = new BrowserBridge();
function detectBridge(): MaxBridge {
  return window.WebApp?.initData
    ? new NativeMaxBridge(window.WebApp)
    : browserBridge;
}
let sdkPromise: Promise<MaxBridge> | undefined;
export function loadMaxBridge(): Promise<MaxBridge> {
  if (sdkPromise) return sdkPromise;
  if (window.WebApp) {
    sdkPromise = Promise.resolve(detectBridge());
    return sdkPromise;
  }
  sdkPromise = new Promise((resolve) => {
    const script = document.createElement("script");
    script.src = "https://st.max.ru/js/max-web-app.js";
    script.async = true;
    script.onload = () => resolve(detectBridge());
    script.onerror = () => resolve(browserBridge);
    document.head.appendChild(script);
  });
  return sdkPromise;
}
export function launchRoute(startParam: string): string | undefined {
  // A fixed allowlist: launch parameters cannot redirect to arbitrary URLs/admin.
  switch (startParam) {
    case "house":
      return "/house";
    case "requests":
      return "/requests";
    case "new_request":
      return "/requests/new";
    default:
      return undefined;
  }
}
