import { UserTie } from '@/core/domain/circles/userTie';

/**
 * User tie service interface
 */

export interface IUserTieService {
    /**
     * Tie users
     */
    tieUsers: (sender: UserTie, receiver: UserTie, circleIds: string[]) => Promise<string>;

    /**
     * Update users tie
     */
    updateUsersTie: (receiverId: string, circleIds: string[]) => Promise<void>;

    /**
     * Remove users' tie
     */
    removeUsersTie: (userId: string) => Promise<void>;

    /**
     * Get user ties
     */
    getUserTies: () => Promise<Array<UserTie>>;

    /**
     * Get the users who tied current user
     */
    getUserTieSender: () => Promise<Array<UserTie>>;
}
