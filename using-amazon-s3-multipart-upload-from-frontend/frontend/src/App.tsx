import { usePresignedUpload } from "./useUploadWithPresignedURL";
import { useSDKUpload } from "./useUploadWithSDK";

function App() {
  const { uploadWithSDK } = useSDKUpload();
  const { uploadWithPresignedURL } = usePresignedUpload();

  return (
    <>
      <p>
        <label htmlFor="sdkInput">Upload with SDK</label>
        <input
          id="sdkInput"
          type="file"
          accept="*"
          onChange={async (e: React.ChangeEvent<HTMLInputElement>) => {
            if (e.target.files && e.target.files.length > 0) {
              await uploadWithSDK(e.target.files[0]);
              e.target.value = "";
            }
          }}
        />
      </p>
      <p>
        <label htmlFor="presignedURLInput">Upload with Presigned URL</label>
        <input
          id="presignedURLInput"
          type="file"
          accept="*"
          onChange={async (e: React.ChangeEvent<HTMLInputElement>) => {
            if (e.target.files && e.target.files.length > 0) {
              await uploadWithPresignedURL(e.target.files[0]);
              e.target.value = "";
            }
          }}
        />
      </p>
    </>
  );
}

export default App;
