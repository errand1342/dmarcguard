import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import { useThemeStore, useSettingsStore } from "./stores";

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

// Initialize stores before mounting
const themeStore = useThemeStore();
const settingsStore = useSettingsStore();

try {
  themeStore.init();
  settingsStore.init();
} catch (error) {
  console.error("Failed to initialize stores:", error);
}

app.mount("#app");