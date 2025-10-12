import { LazyExoticComponent, ReactNode, ComponentType } from 'react';

export interface IPluginRoute {
    path: string;
    component: LazyExoticComponent<ComponentType<any>> | ReactNode;
    exact: boolean;
    private: boolean;
}

export interface IPlugin {
    name: string;
    enabled: boolean;
    baseUrl: string;
    routes: IPluginRoute[];
    components?: ReactNode[];
} 