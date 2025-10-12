import { UserSetting } from '@/core/domain/users/userSetting';
/**
 * User setting interface
 */
export interface IUserSettingService {
    updateUserSetting: (userSetting: UserSetting) => Promise<void>;
    getUserSettings: () => Promise<Record<string, Record<string, any>>>;
}
