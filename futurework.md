# Future Work Notes

A quick list of things that could be improved.

- UI Design
  - Still using basic HTML buttons/text.
- JS code structure
  - Very disorganized, functions could be grouped better for coherency.
- Duplicated code (Both frontend and backend)
  - Utility functions would clean-up the code a lot.

## Bugs (That we know of)
- Minor Bug: Unfinished Proofs
  - Finished Proofs aren't actually removed from the Unfinished Proof list. Minor because a student that has completed the proof will still be labeled as having completed it.
  - Solution: Update record keeping in backend for tracking when a proof is finished. There's too little tracking at the moment to account for unfinished vs finished and removal from dropdown.
- Medium Bug: Test/Quiz/Final Unfinished Problem Return
  - If a student works on a test/quiz/final problem, but do not finish it, then loads another problem, then they will not be able to load the problem again (from their unfinished proof dropdown). They can only load a fresh version from the repository proof dropdown list.
  - Solution is working more on the backend handling. Ideas: Check assignment visibility when a problem has test/quiz/final in the name, and control loading based on that value.
  - NOTE: Another thing that could be worked on in tandem with this issue is that test/quiz/final problems are not loaded *at all* into the finished proof dropdown list. Could make them load only when the assignment is visible.
