import { atom } from "jotai";
import { atomWithStorage } from "jotai/utils";

export const isSidebarExpandedAtom = atom(true);

export interface ActiveSavedReport {
  id: string;
  title: string;
  query: string;
}

export const activeSavedReportsAtom = atomWithStorage<ActiveSavedReport[]>(
  "ce-active-saved-reports",
  [],
);
