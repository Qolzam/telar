/**
 * Simplified Dependency Coordinator
 * 
 * Uses only the simple provider system - no more Inversify complexity!
 * Adapted for Next.js App Router
 */

import { provider } from '@/lib/provider/provider';
import type { ServiceInstance } from '@/lib/provider';
// TODO: Re-enable when data services are migrated
// import { bindHttpService } from '../data/webAPI/dependecyRegisterar';
// import { bindMicros } from '../data/microClient/dependecyRegisterar';
import { SocialProviderTypes } from './socialProviderTypes';

/**
 * Interface for dependency initialization status
 */
interface DependencyStatus {
    coreServicesInitialized: boolean;
    pluginsInitialized: boolean;
}

let dependencyStatus: DependencyStatus = {
    coreServicesInitialized: false,
    pluginsInitialized: false,
};

export const initializeCoreServices = (): void => {
    if (dependencyStatus.coreServicesInitialized) {
        return;
    }

    try {
        // TODO: Re-enable when data services are migrated (Phase 1)
        // Register HTTP service first (required by most other services)
        // const httpRegistered = bindHttpService();
        // if (!httpRegistered) {
        //     throw new Error('Failed to register HTTP service');
        // }

        // Register essential core services (AuthorizeService, etc.)
        // const coreRegistered = bindMicros();
        // if (!coreRegistered) {
        //     throw new Error('Failed to register core services');
        // }

        dependencyStatus.coreServicesInitialized = true;
        console.log('✅ Core services initialized (Phase 0 - plugin system only)');
    } catch (error) {
        console.error('❌ Failed to initialize core services:', error);
        throw new Error('Critical dependency initialization failure');
    }
};

export const initializeDependencyInjection = (): void => {
    try {
        // Initialize core services (HTTP service primarily)
        initializeCoreServices();

        console.log('🚀 Simplified dependency injection system initialized');
    } catch (error) {
        console.error('💥 Critical failure initializing dependency injection:', error);
        throw error;
    }
};

export const initializePluginDependencies = async (): Promise<void> => {
    if (dependencyStatus.pluginsInitialized) {
        return;
    }

    try {
        const isCoreValid = validateCriticalServices();
        if (!isCoreValid) {
            throw new Error('Core services validation failed - cannot initialize plugin dependencies');
        }

        // Plugin dependencies are now automatically registered during plugin loading!
        console.log('✅ Plugin dependency system simplified - dependencies auto-register during plugin loading');
        
        dependencyStatus.pluginsInitialized = true;
        console.log('✅ Plugin dependencies initialized successfully');
    } catch (error) {
        console.error('❌ Failed to initialize plugin dependencies:', error);
        throw new Error('Plugin dependency initialization failure');
    }
};

export const getDependencyStatus = (): DependencyStatus => {
    return { ...dependencyStatus };
};

export const validateCriticalServices = (): boolean => {
    const criticalServices = [
        SocialProviderTypes.HttpService,
    ];

    try {
        const missingServices: string[] = [];

        criticalServices.forEach(serviceType => {
            if (!provider.has(serviceType)) {
                missingServices.push(serviceType);
            }
        });

        if (missingServices.length > 0) {
            console.error('❌ Missing critical services:', missingServices);
            return false;
        }

        console.log('✅ All critical services are available');
        return true;
    } catch (error) {
        console.error('❌ Error validating critical services:', error);
        return false;
    }
};

export const getService = <T extends ServiceInstance>(serviceType: string): T => {
    try {
        return provider.get<T>(serviceType);
    } catch (error) {
        console.error(`❌ Failed to get service ${serviceType}:`, error);
        throw error;
    }
};

export const cleanupDependencyInjection = (): void => {
    try {
        // Clear simple provider
        provider.clear();

        dependencyStatus = {
            coreServicesInitialized: false,
            pluginsInitialized: false,
        };

        console.log('🧹 Dependency injection cleanup completed');
    } catch (error) {
        console.error('❌ Failed to cleanup dependency injection:', error);
    }
};

export const healthCheck = (): { status: 'healthy' | 'degraded' | 'unhealthy'; details: any } => {
    try {
        const status = getDependencyStatus();
        const criticalServicesValid = validateCriticalServices();

        const isHealthy = status.coreServicesInitialized && criticalServicesValid;

        return {
            status: isHealthy ? 'healthy' : 'unhealthy',
            details: {
                dependencyStatus: status,
                criticalServicesValid,
                timestamp: new Date().toISOString(),
            }
        };
    } catch (error) {
        return {
            status: 'unhealthy',
            details: {
                error: (error as Error).message,
                timestamp: new Date().toISOString(),
            }
        };
    }
};
