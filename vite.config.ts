import vinext from "vinext";
import { defineConfig } from "vite";

const pagesBasePath = process.env.PAGES_BASE_PATH || "";

export default defineConfig({
  base: pagesBasePath ? `${pagesBasePath}/` : undefined,
  plugins: [vinext()],
});
