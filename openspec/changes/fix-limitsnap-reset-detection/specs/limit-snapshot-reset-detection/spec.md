## ADDED Requirements

### Requirement: Same-window usage drop triggers high-water reset

When recording a rate-limit sample, the snapshot store SHALL detect an official mid-window usage reset: if the store already contains one or more entries with the same reset epoch as the incoming sample, and the highest stored percentage for that reset epoch exceeds the incoming sample's percentage by more than 5.0 percentage points, the store SHALL delete all existing entries for that reset epoch before recording the incoming sample.

#### Scenario: Official reset detected and old high-water purged

- **WHEN** a sample arrives whose reset epoch matches an existing entry and whose percentage is lower than the stored high-water by more than 5.0 percentage points
- **THEN** all existing entries for that reset epoch are deleted, the new sample is recorded, and a subsequent read returns the new sample's percentage

##### Example: 7d window reset from 75% to 3%

- **GIVEN** the store contains entry `1784311200_075.000`
- **WHEN** a sample with reset epoch 1784311200 and percentage 3.0 is recorded
- **THEN** entry `1784311200_075.000` is deleted, entry `1784311200_003.000` is created, and the resolved high-water percentage is 3.0

#### Scenario: Cross-session jitter within threshold preserves high-water

- **WHEN** a sample arrives whose reset epoch matches an existing entry and whose percentage is lower than the stored high-water by 5.0 percentage points or less
- **THEN** no existing entry is deleted and the resolved high-water percentage remains the stored maximum

##### Example: stale session reports slightly lower reading

- **GIVEN** the store contains entry `1784311200_075.000`
- **WHEN** a sample with reset epoch 1784311200 and percentage 74.5 is recorded
- **THEN** entry `1784311200_075.000` remains, and the resolved high-water percentage is 75.0

### Requirement: Reset detection is scoped to a single reset epoch

Reset detection SHALL compare and delete only entries whose reset epoch equals the incoming sample's reset epoch. Entries with a different reset epoch SHALL NOT be deleted and SHALL NOT participate in the drop comparison; window-transition behavior (a later reset epoch superseding an earlier one at read time) SHALL remain unchanged.

#### Scenario: New window sample does not trigger reset purge

- **WHEN** a sample arrives whose reset epoch differs from all existing entries, regardless of its percentage
- **THEN** no reset purge occurs and existing read-time selection behavior applies unchanged

##### Example: new 7d window opens at low percentage

- **GIVEN** the store contains entry `1784311200_075.000`
- **WHEN** a sample with reset epoch 1784916000 and percentage 2.0 is recorded
- **THEN** entry `1784311200_075.000` is not deleted by reset detection, and entry `1784916000_002.000` is created

### Requirement: Reset purge failures stay silent

Filesystem errors encountered while deleting entries during reset detection SHALL be silently ignored, and the incoming sample SHALL still be recorded. The renderer SHALL NOT emit error text due to reset-detection failures.

#### Scenario: Concurrent render already deleted the entry

- **WHEN** a reset purge attempts to delete an entry that a concurrent render already removed
- **THEN** the deletion error is ignored and the incoming sample is recorded normally
