CREATE TABLE IF NOT EXISTS staff_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id UUID NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_staff_session_staff FOREIGN KEY (staff_id) REFERENCES staff(id) ON DELETE CASCADE,
    CONSTRAINT uq_staff_session_token UNIQUE (token)
);

-- Fast lookup for auth middleware by token
CREATE INDEX IF NOT EXISTS idx_staff_session_token ON staff_session(token);

-- Query/Delete all sessions for a specific staff member
CREATE INDEX IF NOT EXISTS idx_staff_session_staff_id ON staff_session(staff_id);

-- Periodic cleanup for expired sessions
CREATE INDEX IF NOT EXISTS idx_staff_session_expires_at ON staff_session(expires_at);
