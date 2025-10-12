import { Plugin } from '@/plugins/types';
import { AUTH_ROUTES } from './constants/routes';

const authPlugin: Plugin = {
  metadata: {
    id: 'auth',
    name: 'Authentication',
    version: '2.0.0',
    description: 'Authentication and authorization feature - Next.js 15 + React Query implementation',
    priority: 1,
  },
  
  hooks: {
    onLoad: async () => {
      console.log('[Auth Plugin] Loading authentication feature...');
    },
    
    onUnload: async () => {
      console.log('[Auth Plugin] Unloading authentication feature...');
    },
    
    onError: (error: Error) => {
      console.error('[Auth Plugin] Error:', error);
    },
  },
};

export default authPlugin;
export { authPlugin };
