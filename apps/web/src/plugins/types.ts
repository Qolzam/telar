import type { IServiceProvider } from '@/lib/provider';

/**
 * Plugin metadata
 */
export interface PluginMetadata {
  id: string;
  name: string;
  version: string;
  description?: string;
  author?: string;
  dependencies?: string[];
  priority?: number;
}

/**
 * Service registration function
 */
export type ServiceRegistration = (container: IServiceProvider) => void;

/**
 * Plugin lifecycle hooks
 */
export interface PluginHooks {
  onLoad?: () => void | Promise<void>;
  onUnload?: () => void | Promise<void>;
  onError?: (error: Error) => void;
}

/**
 * Plugin component props
 */
export type PluginComponentProps = Record<string, unknown>;

/**
 * Main plugin interface
 */
export interface Plugin {
  metadata: PluginMetadata;
  registerServices?: ServiceRegistration;
  hooks?: PluginHooks;
  routes?: PluginRoute[];
  components?: Record<string, React.ComponentType<PluginComponentProps>>;
}

/**
 * Plugin route definition
 */
export interface PluginRoute {
  path: string;
  component: React.ComponentType<PluginComponentProps>;
  exact?: boolean;
  middleware?: string[];
}

/**
 * Plugin loading result
 */
export interface PluginLoadResult {
  plugin: Plugin;
  loaded: boolean;
  error?: Error;
  loadTime?: number;
}
