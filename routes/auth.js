import express from "express";
import passport from "../auth.js";
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
        info.message || "Authentication failed"
      );
    }
    req.logIn(user, function (err) {
      if (err) {
        return next(err);
      }
      sendResponse(res, 200, user, "Authentication succeeded");
    });
  })(req, res, next);
});

router.post("/logout", function (req, res) {
  req.logout(function (err) {
    if (err) {
      return next(err);
    }
    sendResponse(res, 200, {}, "Logged out successfully");
  });
});

router.get("/auth-status", (req, res) => {
  if (req.isAuthenticated()) {
    sendResponse(
      res,
      200,
      { authenticated: true, user: req.user },
      "User is authenticated"
    );
  } else {
    sendResponse(
      res,
      200,
      { authenticated: false },
      "User is not authenticated"
    );
  }
});

router.use(errorHandler);

export default router;
