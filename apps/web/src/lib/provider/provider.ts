/**
 * Social Engine service provider
 * This module provides dependency injection for the application
 * 
 * NOTE: This is being coordinated with Inversify via dependencyCoordinator.ts
 * for a unified dependency injection system.
 */

/**
 * Base type for all services that can be registered in the container
 */
export type ServiceInstance = object;

/**
 * Service container type - maps service identifiers to service instances
 */
type ServiceContainer = Map<string, ServiceInstance>;

/**
 * Service Provider Interface
 * Defines the contract for dependency injection
 */
export interface IServiceProvider {
  register<T extends ServiceInstance>(serviceType: string, implementation: T): void;
  get<T extends ServiceInstance>(serviceType: string): T;
  has(serviceType: string): boolean;
  clear(): void;
  unregister(serviceType: string): boolean;
  getRegisteredServices(): string[];
}

/**
 * Provider class for dependency injection
 * Implements the IServiceProvider interface
 */
class Provider implements IServiceProvider {
  private services: ServiceContainer = new Map();

  /**
   * Register a service implementation
   */
  register<T extends ServiceInstance>(serviceType: string, implementation: T): void {
    this.services.set(serviceType, implementation);
  }

  /**
   * Get a service implementation by type
   */
  get<T extends ServiceInstance>(serviceType: string): T {
    const service = this.services.get(serviceType);
    if (!service) {
      throw new Error(`Service of type ${serviceType} not registered`);
    }
    return service as T;
  }

  /**
   * Check if a service is registered
   */
  has(serviceType: string): boolean {
    return this.services.has(serviceType);
  }

  /**
   * Clear all registered services
   */
  clear(): void {
    this.services.clear();
  }

  /**
   * Unregister a specific service
   */
  unregister(serviceType: string): boolean {
    return this.services.delete(serviceType);
  }

  /**
   * Get all registered service types
   */
  getRegisteredServices(): string[] {
    return Array.from(this.services.keys());
  }

  async waitForService<T extends ServiceInstance>(serviceType: string, timeoutMs: number = 5000): Promise<T | null> {
    const startTime = Date.now();
    
    while (!this.has(serviceType) && (Date.now() - startTime) < timeoutMs) {
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait 100ms
    }
    
    return this.has(serviceType) ? this.get<T>(serviceType) : null;
  }

  async getServiceSafely<T extends ServiceInstance>(serviceType: string, maxRetries: number = 3, retryDelay: number = 1000): Promise<T | null> {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        if (this.has(serviceType)) {
          return this.get<T>(serviceType);
        }
        
        if (attempt < maxRetries) {
          console.log(`⏳ Service ${serviceType} not available, retrying in ${retryDelay}ms (attempt ${attempt}/${maxRetries})`);
          await new Promise(resolve => setTimeout(resolve, retryDelay));
        }
      } catch (error) {
        if (attempt === maxRetries) {
          console.error(`❌ Failed to get service ${serviceType} after ${maxRetries} attempts:`, error);
          return null;
        }
      }
    }
    
    console.warn(`⚠️ Service ${serviceType} not available after ${maxRetries} attempts`);
    return null;
  }
}

export const provider = new Provider();

// Initialize core dependency injection when this module loads
// This will be coordinated with Inversify via dependencyCoordinator.ts
console.log('📦 Social Engine provider initialized');

// Features on the roadmap
// useAzure(provider)
// userAspNet(provider)
