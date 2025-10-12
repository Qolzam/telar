export type PluginConfigValue = string | number | boolean | PluginConfigObject | PluginConfigArray;
export type PluginConfigObject = { [key: string]: PluginConfigValue };
export type PluginConfigArray = PluginConfigValue[];

export interface PluginConfig {
  id: string;
  enabled: boolean;
  config?: PluginConfigObject;
}

export const enabledPlugins: PluginConfig[] = [
  {
    id: 'test-plugin',
    enabled: true,
    config: {
      debug: true,
    },
  },
  
  {
    id: 'auth',
    enabled: true,
    config: {
      sessionTimeout: 7 * 24 * 60 * 60,
      enableOAuth: false,
    },
  },
];

export function getEnabledPluginIds(): string[] {
  return enabledPlugins
    .filter(p => p.enabled)
    .map(p => p.id);
}

export function getPluginConfig(pluginId: string): PluginConfigObject | undefined {
  const plugin = enabledPlugins.find(p => p.id === pluginId);
  return plugin?.config;
}
