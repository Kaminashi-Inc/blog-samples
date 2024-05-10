import {
	CognitoIdentityClient,
	GetCredentialsForIdentityCommand,
} from "@aws-sdk/client-cognito-identity";
import {
	AbortMultipartUploadCommand,
	CompleteMultipartUploadCommand,
	CreateMultipartUploadCommand,
	S3Client,
	UploadPartCommand,
} from "@aws-sdk/client-s3";
import { v4 as uuidv4 } from "uuid";
import { API_ORIGIN, PART_SIZE } from "./constant";

type GetOpenIDTokenResponse = {
	openIDToken: {
		identityId: string;
		token: string;
	};
	region: string;
	bucket: string;
	keyPrefix: string;
};

export const upload = async (file: File) => {
	const openIDTokenResponse = await fetch(
		`${API_ORIGIN}/openIDToken`,
	).then<GetOpenIDTokenResponse>((response) => response.json());

	const {
		openIDToken: { identityId, token },
		region,
		bucket,
		keyPrefix,
	} = openIDTokenResponse;

	const cognitoClient = new CognitoIdentityClient({ region });
	const getCredentialsCommand = new GetCredentialsForIdentityCommand({
		IdentityId: identityId,
		Logins: {
			"cognito-identity.amazonaws.com": token,
		},
	});
	const cognitoResponse = await cognitoClient.send(getCredentialsCommand);

	const s3Client = new S3Client({
		region,
		credentials: {
			accessKeyId: cognitoResponse.Credentials?.AccessKeyId ?? "",
			secretAccessKey: cognitoResponse.Credentials?.SecretKey ?? "",
			sessionToken: cognitoResponse.Credentials?.SessionToken,
		},
	});

	let uploadId: string | undefined = undefined;
	const objectKey = `${keyPrefix}/${uuidv4()}`;

	try {
		// 1. アップロード開始
		const multipartUpload = await s3Client.send(
			new CreateMultipartUploadCommand({
				Bucket: bucket,
				Key: objectKey,
			}),
		);

		uploadId = multipartUpload.UploadId;

		const uploadPromises: Promise<{
			ETag: string;
			partNumber: number;
		}>[] = [];

		// 2. ファイルを part 毎にアップロード
		for (let i = 0; i < file.size; i += PART_SIZE) {
			const part = file.slice(i, i + PART_SIZE);
			const partNumber = i / PART_SIZE + 1;
			const uploadPartPromise = s3Client
				.send(
					new UploadPartCommand({
						Bucket: bucket,
						Key: objectKey,
						PartNumber: partNumber,
						UploadId: uploadId,
						Body: part,
					}),
				)
				.then((response) => ({
					ETag: response.ETag ?? "",
					partNumber,
				}));
			uploadPromises.push(uploadPartPromise);
		}

		const uploadResponses = await Promise.all(uploadPromises);

		// 3. アップロード完了
		const completion = await s3Client.send(
			new CompleteMultipartUploadCommand({
				Bucket: bucket,
				Key: objectKey,
				UploadId: uploadId,
				MultipartUpload: {
					Parts: uploadResponses.map((response) => ({
						ETag: response.ETag,
						PartNumber: response.partNumber,
					})),
				},
			}),
		);

		console.log({ completion });
	} catch (error) {
		console.error(error);

		if (uploadId) {
			await s3Client.send(
				new AbortMultipartUploadCommand({
					Bucket: bucket,
					Key: objectKey,
					UploadId: uploadId,
				}),
			);
		}
	}
};
