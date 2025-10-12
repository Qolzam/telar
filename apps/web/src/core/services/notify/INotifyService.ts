import { Notification } from '@/core/domain/notifications/notification';

/**
 * Notification service interface
 *
 * @export
 * @interface INotifyService
 */
export interface INotifyService {
    addNotification: (notification: Notification) => Promise<void>;
    getNotifications: () => any;
    deleteNotification: (notificationId: string) => Promise<void>;
    setSeenNotification: (notificationId: string) => Promise<void>;
    setSeenAllNotifications: () => Promise<any>;
}
