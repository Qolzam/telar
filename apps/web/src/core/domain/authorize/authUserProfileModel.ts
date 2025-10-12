import { UserPermissionType } from '@/core/domain/foundation/userPermissionType';

/**
 * Represents the user profile model matching the Go UserProfileModel struct
 */
export class AuthUserProfileModel {
  public objectId: string;
  public fullName: string;
  public socialName: string;
  public avatar: string;
  public banner: string;
  public tagLine: string;
  public created_date: number;
  public last_updated: number;
  public email: string;
  public birthday: number;
  public webUrl: string;
  public companyName: string;
  public voteCount: number;
  public shareCount: number;
  public followCount: number;
  public followerCount: number;
  public postCount: number;
  public facebookId: string;
  public instagramId: string;
  public twitterId: string;
  public accessUserList: string[];
  public permission: UserPermissionType;

  constructor(data: Partial<AuthUserProfileModel> = {}) {
    this.objectId = data.objectId || '';
    this.fullName = data.fullName || '';
    this.socialName = data.socialName || '';
    this.avatar = data.avatar || '';
    this.banner = data.banner || '';
    this.tagLine = data.tagLine || '';
    this.created_date = data.created_date || 0;
    this.last_updated = data.last_updated || 0;
    this.email = data.email || '';
    this.birthday = data.birthday || 0;
    this.webUrl = data.webUrl || '';
    this.companyName = data.companyName || '';
    this.voteCount = data.voteCount || 0;
    this.shareCount = data.shareCount || 0;
    this.followCount = data.followCount || 0;
    this.followerCount = data.followerCount || 0;
    this.postCount = data.postCount || 0;
    this.facebookId = data.facebookId || '';
    this.instagramId = data.instagramId || '';
    this.twitterId = data.twitterId || '';
    this.accessUserList = data.accessUserList || [];
    this.permission = data.permission || UserPermissionType.Public;
  }

  /**
   * Creates a AuthUserProfileModel from a JSON object
   */
  static fromJson(json: any): AuthUserProfileModel {
    return new AuthUserProfileModel({
      objectId: json.objectId || '',
      fullName: json.fullName || '',
      socialName: json.socialName || '',
      avatar: json.avatar || '',
      banner: json.banner || '',
      tagLine: json.tagLine || '',
      created_date: json.created_date || 0,
      last_updated: json.last_updated || 0,
      email: json.email || '',
      birthday: json.birthday || 0,
      webUrl: json.webUrl || '',
      companyName: json.companyName || '',
      voteCount: json.voteCount || 0,
      shareCount: json.shareCount || 0,
      followCount: json.followCount || 0,
      followerCount: json.followerCount || 0,
      postCount: json.postCount || 0,
      facebookId: json.facebookId || '',
      instagramId: json.instagramId || '',
      twitterId: json.twitterId || '',
      accessUserList: json.accessUserList || [],
      permission: json.permission || UserPermissionType.Public
    });
  }

  /**
   * Converts the user profile to a JSON object
   */
  toJson(): object {
    return {
      objectId: this.objectId,
      fullName: this.fullName,
      socialName: this.socialName,
      avatar: this.avatar,
      banner: this.banner,
      tagLine: this.tagLine,
      created_date: this.created_date,
      last_updated: this.last_updated,
      email: this.email,
      birthday: this.birthday,
      webUrl: this.webUrl,
      companyName: this.companyName,
      voteCount: this.voteCount,
      shareCount: this.shareCount,
      followCount: this.followCount,
      followerCount: this.followerCount,
      postCount: this.postCount,
      facebookId: this.facebookId,
      instagramId: this.instagramId,
      twitterId: this.twitterId,
      accessUserList: this.accessUserList,
      permission: this.permission
    };
  }

  /**
   * Gets the primary display name for the user
   */
  getDisplayName(): string {
    return this.fullName || this.socialName || this.email.split('@')[0] || 'Unknown User';
  }

  /**
   * Gets the creation date as a JavaScript Date object
   */
  getCreatedDate(): Date {
    return new Date(this.created_date * 1000); // Convert Unix timestamp to Date
  }

  /**
   * Gets the last updated date as a JavaScript Date object
   */
  getLastUpdatedDate(): Date {
    return new Date(this.last_updated * 1000); // Convert Unix timestamp to Date
  }

  /**
   * Gets the birthday as a JavaScript Date object
   */
  getBirthdayDate(): Date | null {
    return this.birthday ? new Date(this.birthday * 1000) : null;
  }

  /**
   * Checks if the user profile has complete basic information
   */
  isComplete(): boolean {
    return !!(this.fullName && this.email && this.socialName);
  }

  /**
   * Checks if the user has an avatar
   */
  hasAvatar(): boolean {
    return !!(this.avatar && this.avatar.trim() !== '');
  }

  /**
   * Gets the total engagement count (votes + shares)
   */
  getTotalEngagement(): number {
    return this.voteCount + this.shareCount;
  }
}

/**
 * Updated TelarLoginResponse class with the correct UserProfileModel
 */
export class TelarLoginResponse {
  public user: AuthUserProfileModel;
  public accessToken: string;
  public redirect: string;

  constructor(user: AuthUserProfileModel, accessToken: string, redirect: string) {
    this.user = user;
    this.accessToken = accessToken;
    this.redirect = redirect;
  }

  /**
   * Creates a TelarLoginResponse from a JSON object received from the API
   */
  static fromJson(json: any): TelarLoginResponse {
    return new TelarLoginResponse(
      AuthUserProfileModel.fromJson(json.user),
      json.accessToken,
      json.redirect
    );
  }

  /**
   * Converts the response to a JSON object
   */
  toJson(): object {
    return {
      user: this.user.toJson(),
      accessToken: this.accessToken,
      redirect: this.redirect
    };
  }

  /**
   * Validates if the login response contains all required fields
   */
  isValid(): boolean {
    return !!(this.user && this.accessToken && this.redirect);
  }

  /**
   * Returns the user's display name
   */
  getDisplayName(): string {
    return this.user.getDisplayName();
  }

  /**
   * Checks if the user has an avatar
   */
  hasAvatar(): boolean {
    return this.user.hasAvatar();
  }
}