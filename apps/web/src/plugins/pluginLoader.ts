/**
 * Plugin Loader
 * 
 * Handles dynamic import and initialization of plugins.
 */

'use client';

import { Plugin, PluginLoadResult } from './types';
import { enabledPlugins } from './enabledPlugins';

export async function loadPlugin(pluginId: string): Promise<PluginLoadResult> {
  const startTime = performance.now();
  
  try {
    // Dynamic import of the plugin
    // Plugins should export a default object that implements Plugin interface
    const pluginModule = await import(`@/features/${pluginId}/plugin`);
    const plugin: Plugin = pluginModule.default;
    
    if (!plugin.metadata || !plugin.metadata.id) {
      throw new Error(`Invalid plugin: ${pluginId} - Missing metadata`);
    }
    
    if (plugin.hooks?.onLoad) {
      await plugin.hooks.onLoad();
    }
    
    const loadTime = performance.now() - startTime;
    
    return {
      plugin,
      loaded: true,
      loadTime,
    };
  } catch (error) {
    console.error(`Failed to load plugin: ${pluginId}`, error);
    
    return {
      plugin: {
        metadata: {
          id: pluginId,
          name: pluginId,
          version: '0.0.0',
        },
      },
      loaded: false,
      error: error as Error,
      loadTime: performance.now() - startTime,
    };
  }
}

export async function loadAllPlugins(): Promise<PluginLoadResult[]> {
  const enabledIds = enabledPlugins
    .filter(p => p.enabled)
    .map(p => p.id);
  
  console.log(`Loading ${enabledIds.length} plugins...`);
  
  // Load plugins in parallel
  const results = await Promise.all(
    enabledIds.map(id => loadPlugin(id))
  );
  
  // Sort by priority (lower numbers first)
  const sortedResults = results.sort((a, b) => {
    const priorityA = a.plugin.metadata.priority || 100;
    const priorityB = b.plugin.metadata.priority || 100;
    return priorityA - priorityB;
  });
  
  const successCount = sortedResults.filter(r => r.loaded).length;
  const failCount = sortedResults.filter(r => !r.loaded).length;
  
  console.log(`Plugins loaded: ${successCount} succeeded, ${failCount} failed`);
  
  return sortedResults;
}

export function getPlugin(
  loadedPlugins: PluginLoadResult[],
  pluginId: string
): Plugin | undefined {
  const result = loadedPlugins.find(r => r.plugin.metadata.id === pluginId);
  return result?.loaded ? result.plugin : undefined;
}

