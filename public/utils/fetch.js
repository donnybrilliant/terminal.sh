export async function fetchWithTimeout(resource, options = {}, timeout = 8000) {
  const { signal, headers, ...rest } = options;

  // Setup timeout controller
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), timeout);

  const config = {
    method: options.method || "GET",
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    signal: controller.signal,
    ...rest,
  };

  try {
    const response = await fetch(resource, config);
    clearTimeout(id); // Clear the timeout
    const data = await response.json(); // Parse JSON response
    if (!response.ok) {
      throw new Error(data.message || `HTTP error! status: ${response.status}`);
    }
    return data; // Return parsed data
  } catch (error) {
    if (error.name === "AbortError") {
      throw new Error("Request timed out");
    }
    throw error; // Re-throw other errors for the caller to handle
  }
}
