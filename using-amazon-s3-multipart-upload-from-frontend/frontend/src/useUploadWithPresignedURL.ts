import { useCallback } from "react";
import { upload } from "./uploadWithPresignedURL";

export const usePresignedUpload = () => {
	const uploadWithPresignedURL = useCallback(upload, []);

	return {
		uploadWithPresignedURL,
	};
};
