import { BaseDomain } from '@/core/domain/foundation/baseDomain';
import { Comment } from '@/core/domain/comments/comment';
import { PostType } from '@/core/domain/posts/postType';
import { UserPermissionType } from '@/core/domain/foundation/userPermissionType';

/**
 * Post album model for media posts
 */
export interface PostAlbum {
    count: number;
    cover: string;
    coverId: string;
    photos: string[];
    title: string;
}

/**
 * Post interface that matches the API model structure
 */
export interface PostModel {
    objectId: string;
    ownerUserId: string;
    ownerDisplayName: string;
    ownerAvatar: string;
    body: string;
    image?: string;
    imageFullPath?: string;
    video?: string;
    thumbnail?: string;
    album?: PostAlbum;
    score: number;
    commentCounter: number;
    viewCount: number;
    votes: { [userId: string]: boolean };
    tags: string[];
    postTypeId: PostType;
    permission: UserPermissionType;
    accessUserList: string[];
    disableComments: boolean;
    disableSharing: boolean;
    deleted: boolean;
    deletedDate?: number;
    urlKey?: string;
    version?: string;
    created_date: number;
    last_updated: number;
}

/**
 * Post Data Transfer Object (simplified for API responses)
 */
export interface PostDTO extends PostModel {}

/**
 * Legacy Post class - maintained for backward compatibility
 * @deprecated Use PostModel interface instead
 */
export class Post extends BaseDomain {
    constructor(
        /**
         * Post identifier
         */
        public id?: string | null,

        /**
         * The identifier of post type
         */
        public postTypeId?: PostType,

        /**
         * The post creation date
         */
        public creationDate?: number,

        /**
         * The post delete date
         */
        public deleteDate?: number,

        /**
         * The score of post
         */
        public score?: number,

        /**
         * List of voter identifier
         */
        public votes?: { [voterId: string]: boolean },

        /**
         * Post view count
         */
        public viewCount?: number,

        /**
         * Store three last comments to show in slide preview comment
         */
        public comments?: { [commentId: string]: Comment },

        /**
         * The text of post
         */
        public body?: string,

        /**
         * The identifier of post owner
         */
        public ownerUserId?: string,

        /**
         * Full name of post owner
         */
        public ownerDisplayName?: string,

        /**
         * Avatar address of post owner
         */
        public ownerAvatar?: string,

        /**
         * Last post edit date
         */
        public lastEditDate?: number,

        /**
         * Post tags
         */
        public tags?: string[],

        /**
         * Number of comment on the post
         */
        public commentCounter?: number,

        /**
         * The address of image on the post
         */
        public image?: string,

        /**
         * Post image full path
         */
        public imageFullPath?: string,

        /**
         * The address of video on the post
         */
        public video?: string,

        /**
         * The address of video thumbnails on the post
         */
        public thumbnail?: string,

        /**
         * Album data - using new PostAlbum interface
         */
        public album?: PostAlbum,

        /**
         * If writing comment is disabled {true} or not {false}
         */
        public disableComments?: boolean,

        /**
         * If sharing post is disabled {true} or not {false}
         */
        public disableSharing?: boolean,

        /**
         * If the post is deleted {true} or not false
         */
        public deleted?: boolean,

        /**
         * The list of user can access to post
         */
        public accessUserList: Array<string> = [],

        /**
         * User permission type
         */
        public permission: UserPermissionType = UserPermissionType.Public,

        /**
         * Post format version
         */
        public version?: string,

        /**
         * URL key for SEO-friendly URLs
         */
        public urlKey?: string,

        /**
         * Creation timestamp (API field)
         */
        public created_date?: number,

        /**
         * Last update timestamp (API field)
         */
        public last_updated?: number,

        /**
         * Deleted date timestamp
         */
        public deletedDate?: number,
    ) {
        super();
    }

    /**
     * Convert Post class instance to PostModel interface
     */
    toModel(): PostModel {
        return {
            objectId: this.id || '',
            ownerUserId: this.ownerUserId || '',
            ownerDisplayName: this.ownerDisplayName || '',
            ownerAvatar: this.ownerAvatar || '',
            body: this.body || '',
            image: this.image,
            imageFullPath: this.imageFullPath,
            video: this.video,
            thumbnail: this.thumbnail,
            album: this.album,
            score: this.score || 0,
            commentCounter: this.commentCounter || 0,
            viewCount: this.viewCount || 0,
            votes: this.votes || {},
            tags: this.tags || [],
            postTypeId: this.postTypeId || PostType.Text,
            permission: this.permission,
            accessUserList: this.accessUserList,
            disableComments: this.disableComments || false,
            disableSharing: this.disableSharing || false,
            deleted: this.deleted || false,
            deletedDate: this.deletedDate,
            urlKey: this.urlKey,
            version: this.version,
            created_date: this.created_date || this.creationDate || Date.now(),
            last_updated: this.last_updated || this.lastEditDate || Date.now(),
        };
    }

    /**
     * Create Post class instance from PostModel interface
     */
    static fromModel(model: PostModel): Post {
        return new Post(
            model.objectId,
            model.postTypeId,
            model.created_date,
            model.deletedDate,
            model.score,
            model.votes,
            model.viewCount,
            undefined, // comments - not included in API model
            model.body,
            model.ownerUserId,
            model.ownerDisplayName,
            model.ownerAvatar,
            model.last_updated,
            model.tags,
            model.commentCounter,
            model.image,
            model.imageFullPath,
            model.video,
            model.thumbnail,
            model.album,
            model.disableComments,
            model.disableSharing,
            model.deleted,
            model.accessUserList,
            model.permission,
            model.version,
            model.urlKey,
            model.created_date,
            model.last_updated,
            model.deletedDate,
        );
    }
}
