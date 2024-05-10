import { type Hasher, md5 } from "js-md5";
import { API_ORIGIN, PART_SIZE } from "./constant";
import { retry } from "./retry";

type StartMultipartUploadResponse = {
	multipartUploadTarget: {
		uploadId: string;
		bucket: string;
		key: string;
	};
	region: string;
	uploadPartURLInfos: {
		partNumber: number;
		uploadPartURL: string;
	}[];
};

export const upload = async (file: File) => {
	const parts = await convertToParts(file);

	// 1. アップロード開始
	const startMultipartUploadResponse = await fetch(
		`${API_ORIGIN}/startMultipartUpload`,
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				partCount: parts.length,
			}),
		},
	).then<StartMultipartUploadResponse>((response) => response.json());

	const {
		multipartUploadTarget: { uploadId, bucket, key: objectKey },
		uploadPartURLInfos,
	} = startMultipartUploadResponse;

	try {
		const uploadPromises: Promise<{ ETag: string; partNumber: number }>[] =
			[];
		const partChecksums: ArrayBuffer[] = [];

		// 2. ファイルを part 毎にアップロード
		for (let i = 0; i < parts.length; i++) {
			const partContent = parts[i].content;
			const partChecksum = parts[i].checksum;
			const partNumber = uploadPartURLInfos[i].partNumber;
			const uploadPartURL = uploadPartURLInfos[i].uploadPartURL;
			partChecksums.push(partChecksum);

			const uploadPartFetch = () =>
				fetch(`${uploadPartURL}`, {
					method: "PUT",
					headers: {
						"Content-Type": "application/octet-stream",
					},
					body: partContent,
				})
					.then((res) => {
						if (!res.ok) {
							throw new Error("Failed to upload part");
						}
						const eTag = res.headers.get("ETag");
						const expectedETag = `"${buf2hex(partChecksum)}"`;
						if (eTag !== expectedETag) {
							console.warn(
								`ETag mismatch: expected ${expectedETag}, got ${eTag}`,
							);
						}
						return { eTag };
					})
					.then(({ eTag }) => ({
						ETag: eTag ?? "",
						partNumber,
					}));

			uploadPromises.push(retry(uploadPartFetch, 0));
		}

		const uploadResponses = await Promise.all(uploadPromises);

		const checksumOfChecksumsString = calcChecksum(...partChecksums).hex();

		// 3. アップロード完了
		await fetch(`${API_ORIGIN}/completeMultipartUpload`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				uploadId,
				bucket,
				key: objectKey,
				parts: uploadResponses.map((response) => ({
					ETag: response.ETag,
					PartNumber: response.partNumber,
				})),
				checksum: checksumOfChecksumsString,
			}),
		}).then((res) => {
			if (!res.ok) {
				throw new Error("Failed to complete multipart upload");
			}
			return res;
		});
	} catch (error) {
		console.error(error);
		await fetch(`${API_ORIGIN}/abortMultipartUpload`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				uploadId,
				bucket,
				key: objectKey,
			}),
		});
	}
};

const convertToParts = async (file: File) => {
	const parts: {
		content: Blob;
		checksum: ArrayBuffer;
	}[] = [];
	for (let i = 0; i < file.size; i += PART_SIZE) {
		const content = file.slice(i, i + PART_SIZE);
		const checksum = calcChecksum(
			await content.arrayBuffer(),
		).arrayBuffer();
		parts.push({ content, checksum });
	}
	return parts;
};

const buf2hex = (arrayBuffer: ArrayBuffer) => {
	return [...new Uint8Array(arrayBuffer)]
		.map((x) => x.toString(16).padStart(2, "0"))
		.join("");
};

const calcChecksum = (...buffer: ArrayBuffer[]): Hasher => {
	const hash = md5.create();
	for (const b of buffer) {
		hash.update(b);
	}
	return hash;
};
