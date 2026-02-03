import type { Plugin } from "@opencode-ai/plugin"
import { appendFile } from "node:fs/promises"

const LOG_FILE = "/tmp/crew-opencode-plugin.log"

const log = async (message: string) => {
  const timestamp = new Date().toISOString()
  await appendFile(LOG_FILE, `[${timestamp}] ${message}\n`).catch(() => { })
}

export const CrewHooksPlugin: Plugin = async ({ $ }) => {

  const updateSubstate = async (substate: string) => {
    try {
      await $`crew edit {{.TaskID}} ${substate}`;
    } catch {
      // Ignore failures to avoid breaking hook execution
    }
  };

  return {
    event: async ({ event }) => {
      await log(`event: ${JSON.stringify(event)}`)

      // Check if current status is in_review (if so, skip auto-switching)
      const isInReview = async () => {
        try {
          const { json } = await $`crew show {{.TaskID }} --json`;
          return json().status === "done";
        } catch {
          return false;
        }
      };

      // Transition to needs_input: session idle
      if (event.type === "session.idle") {
        if (!(await isInReview())) {
          await updateSubstate("awaiting_user");
        }
      }

      // Transition to needs_input or in_progress: session status change
      if (event.type === "session.status") {
        if (!(await isInReview())) {
          if (event.properties.status.type === "idle") {
            await updateSubstate("awaiting_user");
          } else if (event.properties.status.type === "busy") {
            await updateSubstate("running");
          }
        }
      }

      // Transition to needs_input: permission asked
      if (event.type === "permission.updated") {
        if (!(await isInReview())) {
          await updateSubstate("awaiting_permission");
        }
      }
    }
  }
}
