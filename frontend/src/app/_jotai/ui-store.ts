import { atom } from "jotai";

export const isSidebarExpandedAtom = atom(true);

export interface ActiveSavedReport {
  id: string;
  title: string;
  query: string;
}

export const activeSavedReportsAtom = atom<ActiveSavedReport[]>([]);
