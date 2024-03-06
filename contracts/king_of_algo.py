import os 
from pyteal import *

"""King Of Algo"""

king_address_key = Bytes("king")
king_price_key = Bytes("king_price")
init_price_key = Bytes("init_price")
end_of_reign_timestamp_key = Bytes("end_of_reign_timestamp")
admin_address_key = Bytes("admin")
reign_period_key = Bytes("reign_period")
admin_fee_key = Bytes("admin_fee")
reward_multiplier_key = Bytes("reward_multiplier")

empty_str = Bytes("")
initial_king_price = Int(100000) # Has to be > 0
admin_fee_fp = Int(5) # 0.05 with 1/100 scaling.
reward_multiplier_fp = Int(75) # 0.75 with 1/100 scaling
fixed_point_scaling = Int(100)
king_price_multiplier = Int(2)

def approval_program():
    handle_optin = Reject()
    handle_closeout = Reject()

    program = Cond(
        [Txn.application_id() == Int(0), handle_creation()],
        [Txn.on_completion() == OnComplete.OptIn, handle_optin],
        [Txn.on_completion() == OnComplete.CloseOut, handle_closeout],
        [Txn.on_completion() == OnComplete.UpdateApplication, allow_when_admin()],
        [Txn.on_completion() == OnComplete.DeleteApplication, allow_when_admin()],
        [Txn.on_completion() == OnComplete.NoOp, handle_claim()]
    )
    return compileTeal(program, Mode.Application, version=6)

# allow update if the sender is the admin
def allow_when_admin() -> Expr:
    return Seq(
        Assert(Txn.sender() == App.globalGet(admin_address_key)),
        Approve()
    )

def handle_creation() -> Expr:
    return Seq(
        set_admin(),
        set_admin_fee(),
        set_period(),
        set_reward_multiplier(),
        set_init_state(),
        Approve()
    )

def handle_claim() -> Expr:
    return Cond(
        [App.globalGet(king_address_key) == empty_str, handle_when_is_the_first_king()],
        [App.globalGet(king_address_key) != empty_str, handle_when_king_is_set()]
    )

def handle_when_is_the_first_king() -> Expr:
    appTx = Gtxn[0]
    adminFeeTx = Gtxn[1]
    compensationTx = Gtxn[2]

    return Cond(
        [
            And(
                validate_tx_group(3),
                validate_app_tx(appTx),
                validate_admin_fee_tx(adminFeeTx),
                validate_compensation_tx(compensationTx),
                validate_amounts_for_first_king(adminFeeTx, compensationTx),
            ),
            Seq(
                reset_timestamp(),
                set_new_king(),
                Approve()
            )
        ]
    )

def handle_when_king_is_set() -> Expr:
    appTx = Gtxn[0]
    adminFeeTx = Gtxn[1]
    compensationTx = Gtxn[2]
    rewardTx = Gtxn[3]

    return Cond(
        [
            And(
                validate_tx_group(4),
                validate_app_tx(appTx),
                validate_admin_fee_tx(adminFeeTx),
                validate_compensation_tx(compensationTx),
                validate_reward_tx(rewardTx),
            ),
            Seq(
                If(App.globalGet(end_of_reign_timestamp_key) > Global.latest_timestamp())
                .Then(
                    assert_overthrowing_amounts(adminFeeTx, compensationTx, rewardTx),
                )
                .Else(
                    assert_fee_for_inner_tx(),
                    assert_end_of_reign_amounts(adminFeeTx, compensationTx, rewardTx),
                    send_compensation_to_the_dead_king(rewardTx.receiver(), get_last_king_compensation()),
                    set_init_state(),
                ),
                set_new_king(),
                Approve()
            )
        ]
    )

def validate_reward_tx(tx: TxnObject) -> Expr:
    return Seq(
        Assert(tx.type_enum() == TxnType.Payment),
        Assert(tx.sender() != App.globalGet(king_address_key)),
        Assert(tx.receiver() == App.globalGet(king_address_key)),
        Assert(tx.close_remainder_to() == Global.zero_address()),
        Int(1),
    )

def validate_app_tx(tx: TxnObject) -> Expr:
    return tx.type_enum() == TxnType.ApplicationCall


def validate_compensation_tx(tx: TxnObject) -> Expr:
    return Seq(
        Assert(tx.type_enum() == TxnType.Payment),
        Assert(tx.sender() != App.globalGet(king_address_key)),
        Assert(tx.receiver() == Global.current_application_address()),
        Assert(tx.close_remainder_to() == Global.zero_address()),
        Int(1),
    )

def validate_admin_fee_tx(tx: TxnObject) -> Expr:
    return Seq(
        Assert(tx.type_enum() == TxnType.Payment),
        Assert(tx.sender() != App.globalGet(king_address_key)),
        Assert(tx.receiver() == App.globalGet(admin_address_key)),
        Assert(tx.close_remainder_to() == Global.zero_address()),
        Int(1),
    )

def validate_tx_group(num_of_transactions) -> Expr:
    return Seq(
        Assert(Global.group_size() == Int(num_of_transactions)),
        Assert(Txn.group_index() == Int(0)),
        # https://developer.algorand.org/docs/get-details/dapps/avm/teal/guidelines/,
        Assert(check_rekey_zero(num_of_transactions)),
        Int(1),
    )

def validate_amounts_for_first_king(adminFeeTx: TxnObject, compensationTx: TxnObject) -> Expr:
    return Seq(
            Assert(adminFeeTx.amount() + compensationTx.amount() == App.globalGet(init_price_key)),
            Assert(adminFeeTx.amount() == calculate_admin_fee_from_price(App.globalGet(init_price_key))),
            Int(1),
        )

def assert_overthrowing_amounts(adminFeeTx: TxnObject, compensationTx: TxnObject, rewardTx: TxnObject) -> Expr:
    return Seq(
            Assert(adminFeeTx.amount() + compensationTx.amount() + rewardTx.amount() == App.globalGet(king_price_key)),
            Assert(rewardTx.amount() == calculate_reward_from_price(App.globalGet(king_price_key))),
            Assert(adminFeeTx.amount() == calculate_admin_fee_from_price(App.globalGet(king_price_key))),
        )

def assert_end_of_reign_amounts(adminFeeTx: TxnObject, compensationTx: TxnObject, rewardTx: TxnObject) -> Expr:
    return Seq(
            Assert(adminFeeTx.amount() + compensationTx.amount() + rewardTx.amount() == App.globalGet(init_price_key)),
            Assert(rewardTx.amount() == calculate_reward_from_price(App.globalGet(init_price_key))),
            Assert(adminFeeTx.amount() == calculate_admin_fee_from_price(App.globalGet(init_price_key))),
        )

@Subroutine(TealType.none)
def set_new_king() -> Expr:
    scratchPrice = ScratchVar(TealType.uint64)

    return Seq(
        scratchPrice.store(App.globalGet(king_price_key)),
        App.globalPut(king_price_key, scratchPrice.load()* king_price_multiplier),
        App.globalPut(king_address_key, Gtxn[2].sender()),
    )

@Subroutine(TealType.none)
def set_init_state() -> Expr:
    return Seq(
        App.globalPut(king_address_key, empty_str),
        App.globalPut(init_price_key, initial_king_price),
        App.globalPut(king_price_key, initial_king_price), # Has to be > 0
        reset_timestamp(),
    )


@Subroutine(TealType.none)
def reset_timestamp() -> Expr:
    return Seq(App.globalPut(end_of_reign_timestamp_key, Global.latest_timestamp() + App.globalGet(reign_period_key)))  # End reign timestamp.

def set_admin() -> Expr:
    return App.globalPut(admin_address_key, Txn.sender())

def set_admin_fee() -> Expr:
    # TODO: could be parameter.
    return App.globalPut(admin_fee_key, admin_fee_fp)

def set_reward_multiplier() -> Expr:
    # TODO: could be parameter.
    return App.globalPut(reward_multiplier_key, reward_multiplier_fp)

def set_period() -> Expr:
    return App.globalPut(reign_period_key, Btoi(Txn.application_args[0]))

def assert_fee_for_inner_tx() -> Expr:
    """The new king has to pay for the tx fee of the inner tx to the previous king"""
    return Seq(   
        Assert(Txn.fee() == Global.min_txn_fee() * Int(2)),
    )

def send_compensation_to_the_dead_king(receiver: TxnExpr, amount: Int) -> Expr:
    return Seq(
        InnerTxnBuilder.Begin(),
        InnerTxnBuilder.SetFields(
            {
                TxnField.type_enum: TxnType.Payment,
                TxnField.receiver: receiver, # When you do an inner tx the address must also be in the foreign account array.
                TxnField.amount: amount,
                TxnField.fee: Int(0),
            }
        ),
        InnerTxnBuilder.Submit(),
    )

def get_last_king_compensation() -> Expr:
    return Minus(Balance(Global.current_application_address()), MinBalance(Global.current_application_address()))

def calculate_admin_fee_from_price(amount) -> Expr:
    return mutiply_fixed_point(amount, App.globalGet(admin_fee_key), fixed_point_scaling)

def calculate_reward_from_price(amount) -> Expr:
    return mutiply_fixed_point(amount, App.globalGet(reward_multiplier_key), fixed_point_scaling)

@Subroutine(TealType.uint64)
def mutiply_fixed_point(a, fixed_point, scaling) -> Expr:
    """Returns the result of multiplication to a fixed_point divided by the scaling
    Args:
        a: uint64 multiplier
        fixed_point: uint64 multiplicand
    Returns:
        uint64 result of the multiplication
    """
    return div_ceil(Mul(a, fixed_point), scaling)

@Subroutine(TealType.uint64)
def div_ceil(a, b) -> Expr:
    """Returns the result of division rounded up to the next integer
    Args:
        a: uint64 numerator for the operation
        b: uint64 denominator for the operation
    Returns:
        uint64 result of a truncated division + 1
    """
    q = a / b
    return If(a % b > Int(0), q + Int(1), q)

def check_rekey_zero(num_transactions) -> Expr:
    return And(*[
                Gtxn[i].rekey_to() == Global.zero_address()
                for i in range(num_transactions)
            ]
        )

def clear_state_program():
   program = Approve()
   return compileTeal(program, Mode.Application, version=6)

if __name__ == "__main__":
    path = os.path.dirname(os.path.abspath(__file__))

    with open(os.path.join(path,"approval.teal"), "w") as f:
        f.write(approval_program())

    with open(os.path.join(path, "clear.teal"), "w") as f:
        f.write(clear_state_program())