/**
 * Test Plugin
 * 
 * A minimal plugin to verify the plugin system works.
 */

import { Plugin } from '@/plugins/types';

const testPlugin: Plugin = {
  metadata: {
    id: 'test-plugin',
    name: 'Test Plugin',
    version: '1.0.0',
    description: 'A test plugin to verify the plugin system',
    priority: 1, // Load first
  },
  
  registerServices: () => {
    console.log('[Test Plugin] Registering services...');
    // Register services with the DI container when needed
    // Example: container.register('testService', new TestService());
  },
  
  hooks: {
    onLoad: () => {
      console.log('[Test Plugin] Plugin loaded successfully! ✅');
    },
    
    onUnload: () => {
      console.log('[Test Plugin] Plugin unloading...');
    },
    
    onError: (error) => {
      console.error('[Test Plugin] Error:', error);
    },
  },
  
  components: {},
};

export default testPlugin;
