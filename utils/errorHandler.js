// utils/errorHandler.js
export default function errorHandler(err, req, res, next) {
  console.error(err.stack); // Log the stack trace for debugging

  const statusCode = err.statusCode || 500;
  const errorMessage = err.message || "Internal Server Error";

  res.status(statusCode).json({
    success: false,
    message: errorMessage,
  });
}
