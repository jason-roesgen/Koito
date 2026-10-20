import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, loadEnv } from "vite";
import { demoDataPlugin } from "./demo";
import tsconfigPaths from "vite-tsconfig-paths";
import { vanillaExtractPlugin } from "@vanilla-extract/vite-plugin";

const isDocker = process.env.BUILD_TARGET === "docker";

export default defineConfig(({ mode }) => {
  const demo = loadEnv(mode, process.cwd(), "VITE_").VITE_KOITO_DEMO === "true";
  return {
  plugins: [...(demo ? [demoDataPlugin()] : []), tailwindcss(), reactRouter(), vanillaExtractPlugin()],
  server: {
    proxy: {
      "/apis": {
        target: "http://localhost:4110",
        changeOrigin: true,
      },
      "/image": {
        target: "http://localhost:4110",
        changeOrigin: true,
      },
    },
  },
  resolve: {
    tsconfigPaths: true,
    alias: {
      ...(isDocker ? { "react-dom/server": "react-dom/server.node" } : {}),
    },
  },
  };
});
