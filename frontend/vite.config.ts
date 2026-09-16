import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
  base: "/max/",
  plugins: [react()],
  server: {
    proxy: {
      "/max/api": {
        target: "http://localhost:8080",
        rewrite: (path) => path.replace(/^\/max/, ""),
      },
    },
  },
});
