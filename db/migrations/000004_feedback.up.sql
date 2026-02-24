-- ─── Feedback / Suggestion Box ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS feedback (
  id          BIGSERIAL PRIMARY KEY,
  user_email  TEXT NOT NULL,
  user_role   TEXT NOT NULL DEFAULT 'guest',  -- guest, subscriber, admin
  category    TEXT NOT NULL DEFAULT 'saran',  -- saran, bug, pertanyaan
  rating      SMALLINT,                       -- 1-5 (emoji scale)
  message     TEXT NOT NULL,
  page_url    TEXT,                           -- which page the feedback was sent from
  user_agent  TEXT,
  ip          INET,
  is_read     BOOLEAN DEFAULT FALSE,
  created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_feedback_created ON feedback(created_at DESC);
CREATE INDEX idx_feedback_category ON feedback(category);
CREATE INDEX idx_feedback_is_read ON feedback(is_read);
