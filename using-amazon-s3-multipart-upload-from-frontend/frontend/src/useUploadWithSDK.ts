import { useCallback } from "react";
import { upload } from "./uploadWithSDK";

export const useSDKUpload = () => {
	const uploadWithSDK = useCallback(upload, []);

	return {
		uploadWithSDK,
	};
};
