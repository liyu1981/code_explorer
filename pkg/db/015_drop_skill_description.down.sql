-- Add back description column to agent_skills
ALTER TABLE agent_skills ADD COLUMN description TEXT NOT NULL DEFAULT '';
