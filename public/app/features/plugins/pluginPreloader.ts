import { AppPluginConfig } from '@grafana/runtime';

import { setExtensionItemCallback } from './extensions/registry';
import { importPluginModule } from './plugin_loader';

export async function preloadPlugins(apps: Record<string, AppPluginConfig> = {}): Promise<void> {
  const pluginsToPreload = Object.values(apps).filter((app) => app.preload);
  await Promise.all(pluginsToPreload.map(preloadPlugin));
}

async function preloadPlugin(plugin: AppPluginConfig): Promise<void> {
  const { path, version, id } = plugin;
  try {
    const { plugin } = await importPluginModule(path, version);
    setExtensionItemCallback(id, plugin);
  } catch (error: unknown) {
    console.error(`Failed to load plugin: ${path} (version: ${version})`, error);
  }
}
