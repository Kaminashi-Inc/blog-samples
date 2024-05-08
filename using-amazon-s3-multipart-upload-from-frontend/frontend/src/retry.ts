export const sleep = (ms: number) =>
	new Promise((resolve) => setTimeout(resolve, ms));

export const retry = async <T>(
	fn: () => Promise<T>,
	retryCount = 5,
): Promise<T> => {
	const _retry = async (currentRetry: number): Promise<T> => {
		try {
			return await fn();
		} catch (error) {
			if (currentRetry === retryCount) {
				throw error;
			}
			await sleep(2 ** currentRetry * 1000);
			return _retry(currentRetry + 1);
		}
	};
	return await _retry(0);
};
