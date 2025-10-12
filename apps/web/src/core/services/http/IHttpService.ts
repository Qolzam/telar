
export interface IHttpService {
    /**
     * Perform a GET request
     * @param url The URL to request
     * @param params Optional query parameters
     */
    get<T = any>(url: string, params?: Record<string, any>): Promise<T>;

    /**
     * Perform a POST request
     * @param url The URL to request
     * @param data The data to send
     * @param params Optional query parameters
     */
    post<T = any>(url: string, data?: any, params?: Record<string, any>): Promise<T>;

    /**
     * Perform a PUT request
     * @param url The URL to request
     * @param data The data to send
     * @param params Optional query parameters
     */
    put<T = any>(url: string, data?: any, params?: Record<string, any>): Promise<T>;

    /**
     * Perform a DELETE request
     * @param url The URL to request
     * @param params Optional query parameters
     */
    delete<T = any>(url: string, params?: Record<string, any>): Promise<T>;

    /**
     * Perform a file upload
     * @param url The URL to request
     * @param file The file to upload
     * @param onProgress Progress callback
     * @param params Optional query parameters
     */
    uploadFile<T = any>(
        url: string,
        file: File,
        onProgress?: (progress: number) => void,
        params?: Record<string, any>,
    ): Promise<T>;
}
