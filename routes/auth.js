import express from "express";
import passport from "../auth.js";
const router = express.Router();

router.post("/login", function (req, res, next) {
  passport.authenticate("local", function (err, user, info) {
    if (err) {
      return next(err);
    }
    if (!user) {
      return res
        .status(400)
        .json({ success: false, message: "Authentication failed" });
    }
    req.logIn(user, function (err) {
      if (err) {
        return next(err);
      }
      res.json({
        success: true,
        message: "Authentication succeeded",
        user: user,
      });
    });
  })(req, res, next);
});

router.post("/logout", function (req, res) {
  req.logout(function (err) {
    if (err) {
      console.log(err);
      return res.status(500).json({ message: "Logout failed", error: err });
    }
    res.json({ message: "Logged out successfully" });
  });
});

router.get("/auth-status", (req, res) => {
  if (req.isAuthenticated()) {
    res.json({ authenticated: true, user: req.user });
  } else {
    res.json({ authenticated: false });
  }
});

export default router;
