'use client';

import { 
  createContext, 
  useContext, 
  useEffect, 
  useState, 
  ReactNode
} from 'react';
import { Plugin } from './types';
import { loadAllPlugins } from './pluginLoader';
import { provider } from '@/lib/provider';

interface PluginContextType {
  plugins: Plugin[];
  isLoading: boolean;
  isReady: boolean;
  errors: Error[];
  getPlugin: (id: string) => Plugin | undefined;
}

const PluginContext = createContext<PluginContextType | undefined>(undefined);

interface PluginProviderProps {
  children: ReactNode;
}

export function PluginProvider({ children }: PluginProviderProps) {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isReady, setIsReady] = useState(false);
  const [errors, setErrors] = useState<Error[]>([]);

  useEffect(() => {
    let mounted = true;

    async function initializePlugins() {
      console.log('[PluginProvider] Initializing plugin system...');
      
      try {
        const results = await loadAllPlugins();
        
        if (!mounted) return;
        
        const loadedPlugins = results
          .filter(r => r.loaded)
          .map(r => r.plugin);
        
        const loadErrors = results
          .filter(r => !r.loaded && r.error)
          .map(r => r.error!);
        
        for (const plugin of loadedPlugins) {
          if (plugin.registerServices) {
            try {
              plugin.registerServices(provider);
              console.log(`[PluginProvider] Registered services for: ${plugin.metadata.name}`);
            } catch (error) {
              console.error(`[PluginProvider] Failed to register services for: ${plugin.metadata.name}`, error);
              loadErrors.push(error as Error);
            }
          }
        }
        
        setPlugins(loadedPlugins);
        setErrors(loadErrors);
        setIsReady(true);
        
        console.log(`[PluginProvider] Ready! ${loadedPlugins.length} plugins loaded.`);
      } catch (error) {
        console.error('[PluginProvider] Fatal error during initialization:', error);
        setErrors([error as Error]);
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    }

    initializePlugins();

    // Cleanup function runs on component unmount
    // Note: plugins is intentionally not in deps array - we only initialize once on mount
    // The cleanup will use whatever plugins are loaded at unmount time
    return () => {
      mounted = false;
      
      plugins.forEach(plugin => {
        if (plugin.hooks?.onUnload) {
          try {
            plugin.hooks.onUnload();
          } catch (error) {
            console.error(`[PluginProvider] Error during plugin unload: ${plugin.metadata.name}`, error);
          }
        }
      });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const getPlugin = (id: string): Plugin | undefined => {
    return plugins.find(p => p.metadata.id === id);
  };

  const contextValue: PluginContextType = {
    plugins,
    isLoading,
    isReady,
    errors,
    getPlugin,
  };

  return (
    <PluginContext.Provider value={contextValue}>
      {children}
    </PluginContext.Provider>
  );
}

export function usePlugins(): PluginContextType {
  const context = useContext(PluginContext);
  
  if (!context) {
    throw new Error('usePlugins must be used within PluginProvider');
  }
  
  return context;
}

export function usePlugin(pluginId: string): Plugin | undefined {
  const { getPlugin } = usePlugins();
  return getPlugin(pluginId);
}

export function usePluginsReady(): boolean {
  const { isReady } = usePlugins();
  return isReady;
}
