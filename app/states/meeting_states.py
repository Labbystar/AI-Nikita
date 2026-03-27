from aiogram.fsm.state import State, StatesGroup


class CreateMeetingState(StatesGroup):
    waiting_for_title = State()
    waiting_for_description = State()
    waiting_for_source_type = State()


class AddParticipantState(StatesGroup):
    waiting_for_name = State()


class AddItemState(StatesGroup):
    waiting_for_content = State()


class AddActionState(StatesGroup):
    waiting_for_action = State()
