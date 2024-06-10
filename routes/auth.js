import express from "express";
import passport, { generateToken } from "../auth.js";
import { sendResponse } from "../utils/responseUtils.js";
import errorHandler from "../utils/errorHandler.js";

const router = express.Router();

router.post("/login", function (req, res, next) {
  passport.authenticate("local", function (err, user, info) {
    if (err) {
      return next(err); // Errors will be caught by errorHandler
    }
    if (!user) {
      return sendResponse(
        res,
        401,
        {},
        info?.message || "Authentication failed"
      );
    }
    const token = generateToken(user);
    sendResponse(res, 200, { token, user }, "Authentication succeeded");
  })(req, res, next);
});

router.use(errorHandler);

export default router;
