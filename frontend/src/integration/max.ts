// Replace this adapter when enabling the official MAX Bridge.
export interface MaxBridge {
  ready(): void;
  platform: "mock" | "max";
}
export class MockMaxBridge implements MaxBridge {
  platform = "mock" as const;
  ready() {
    /* Web preview and hackathon demo need no external calls. */
  }
}
export const maxBridge: MaxBridge = new MockMaxBridge();
