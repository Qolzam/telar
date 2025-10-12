import { Circle } from '@/core/domain/circles/circle';

/**
 * Circle service interface
 *
 * @export
 * @interface ICircleService
 */
export interface ICircleService {
    addCircle: (circle: { name: string }) => Promise<string>;
    updateCircle: (circleId: string, circle: { name: string }) => Promise<void>;
    deleteCircle: (circleId: string) => Promise<void>;
    getCircles: (userId: string) => Promise<{
        [circleId: string]: Circle;
    }>;
}
