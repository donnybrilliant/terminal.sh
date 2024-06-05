// utils/errorHandler.js

import winston from "winston";

const logger = winston.createLogger({
  transports: [
    new winston.transports.Console(),
    new winston.transports.File({ filename: "error.log" }),
  ],
});

export default function errorHandler(err, req, res, next) {
  //console.error(err.stack); // Log the stack trace for debugging
  logger.error(err.stack);

  const statusCode = err.statusCode || 500;
  const errorMessage = err.message || "Internal Server Error";

  res.status(statusCode).json({
    success: false,
    message: errorMessage,
  });
}
